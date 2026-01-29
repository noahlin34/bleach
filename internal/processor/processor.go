package processor

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"

	"bleach/pkg/imgutil"
)

func Run(ctx context.Context, root string, opts Options, updates chan<- ProgressUpdate) (Summary, []ScanReport, error) {
	summary := Summary{}
	var reports []ScanReport

	info, err := os.Stat(root)
	if err != nil {
		return summary, nil, err
	}

	absRoot, err := filepath.Abs(root)
	if err != nil {
		return summary, nil, err
	}

	var outputAbs string
	var outputInsideRoot bool
	if opts.Mode == ModeClean && !opts.InPlace && opts.OutputDir != "" {
		if absOut, outErr := filepath.Abs(opts.OutputDir); outErr == nil {
			outputAbs = absOut
			absRootClean := filepath.Clean(absRoot)
			outputClean := filepath.Clean(outputAbs)
			if outputClean != absRootClean && isWithin(outputClean, absRootClean) {
				outputInsideRoot = true
			}
		}
	}

	jobs := make(chan Job)
	results := make(chan Result)

	workers := runtime.NumCPU()
	var wg sync.WaitGroup
	wg.Add(workers)
	for i := 0; i < workers; i++ {
		go func() {
			defer wg.Done()
			worker(ctx, jobs, results, opts, updates)
		}()
	}

	collectorDone := make(chan struct{})
	go func() {
		defer close(collectorDone)
		for res := range results {
			if res.Supported {
				summary.Total++
				summary.Processed++
				if updates != nil {
					updates <- ProgressUpdate{ProcessedDelta: 1}
				}
			}
			if res.Err != nil {
				summary.Errors++
				if updates != nil {
					updates <- ProgressUpdate{ErrorDelta: 1}
				}
			}
			if res.Leaks > 0 {
				summary.Leaks += res.Leaks
				if updates != nil {
					updates <- ProgressUpdate{LeakDelta: res.Leaks}
				}
			}
			if res.BytesSaved != 0 {
				summary.BytesSaved += res.BytesSaved
				if updates != nil {
					updates <- ProgressUpdate{BytesSavedDelta: res.BytesSaved}
				}
			}
			if len(res.Report) > 0 || res.Supported {
				reports = append(reports, ScanReport{Path: res.Display, Categories: res.Report})
			}
		}
	}()

	producerErr := make(chan error, 1)
	go func() {
		defer close(jobs)

		sendJob := func(job Job) error {
			if ctx == nil {
				jobs <- job
				return nil
			}
			select {
			case jobs <- job:
				return nil
			case <-ctx.Done():
				return ctx.Err()
			}
		}

		if !info.IsDir() {
			job := Job{
				Path:    absRoot,
				RelPath: filepath.Base(absRoot),
				Display: filepath.Base(absRoot),
			}
			producerErr <- sendJob(job)
			return
		}

		fsys := os.DirFS(absRoot)
		err := fs.WalkDir(fsys, ".", func(path string, d fs.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			if d.IsDir() {
				if outputInsideRoot {
					fullDir := filepath.Join(absRoot, path)
					if isWithin(fullDir, outputAbs) {
						return fs.SkipDir
					}
				}
				return nil
			}
			if !d.Type().IsRegular() {
				return nil
			}

			fullPath := filepath.Join(absRoot, path)
			display := path

			if err := sendJob(Job{
				Path:    fullPath,
				RelPath: path,
				Display: display,
			}); err != nil {
				return err
			}
			return nil
		})
		producerErr <- err
	}()

	wg.Wait()
	close(results)
	<-collectorDone

	if err := <-producerErr; err != nil {
		return summary, reports, err
	}

	if ctx != nil {
		if err := ctx.Err(); err != nil && !errors.Is(err, context.Canceled) {
			return summary, reports, err
		}
	}

	return summary, reports, nil
}

func worker(ctx context.Context, jobs <-chan Job, results chan<- Result, opts Options, updates chan<- ProgressUpdate) {
	for job := range jobs {
		if ctx != nil {
			if err := ctx.Err(); err != nil {
				return
			}
		}

		res := Result{Path: job.Path, RelPath: job.RelPath, Display: job.Display}

		file, err := os.Open(job.Path)
		if err != nil {
			res.Err = err
			results <- res
			continue
		}

		kind, err := imgutil.SniffReader(file)
		if err != nil {
			_ = file.Close()
			res.Err = err
			results <- res
			continue
		}

		if kind == imgutil.KindUnknown {
			_ = file.Close()
			continue
		}

		res.Supported = true
		if updates != nil {
			updates <- ProgressUpdate{TotalDelta: 1}
		}

		switch opts.Mode {
		case ModeScan:
			report, err := scanFile(file, kind)
			_ = file.Close()
			if err != nil {
				res.Err = err
				results <- res
				continue
			}
			res.Report = report
		case ModeClean:
			leaks, err := countLeaks(file, kind)
			if err != nil {
				_ = file.Close()
				res.Err = err
				results <- res
				continue
			}
			res.Leaks = leaks
			if _, err := file.Seek(0, io.SeekStart); err != nil {
				_ = file.Close()
				res.Err = err
				results <- res
				continue
			}

			saved, err := cleanFile(file, job, kind, opts)
			_ = file.Close()
			if err != nil {
				res.Err = err
				results <- res
				continue
			}
			res.BytesSaved = saved
		default:
			_ = file.Close()
			res.Err = fmt.Errorf("unknown mode")
			results <- res
			continue
		}

		results <- res
	}
}

func scanFile(file *os.File, kind imgutil.Kind) ([]string, error) {
	switch kind {
	case imgutil.KindJPEG, imgutil.KindTIFF:
		analysis, err := analyzeExif(file)
		if err != nil {
			return nil, err
		}
		return categoriesFromExif(analysis), nil
	case imgutil.KindPNG:
		analysis, err := scanPNGMetadata(file)
		if err != nil {
			return nil, err
		}
		return categoriesFromPNG(analysis), nil
	default:
		return nil, nil
	}
}

func categoriesFromExif(analysis ExifAnalysis) []string {
	cats := []string{}
	if analysis.HasGPS {
		cats = append(cats, "GPS")
	}
	if analysis.HasModel {
		cats = append(cats, "Device Model")
	}
	if analysis.HasTimestamp {
		cats = append(cats, "Timestamp")
	}
	return cats
}

func categoriesFromPNG(analysis PngAnalysis) []string {
	cats := []string{}
	if analysis.HasGPS {
		cats = append(cats, "GPS")
	}
	if analysis.HasModel {
		cats = append(cats, "Device Model")
	}
	if analysis.HasTimestamp {
		cats = append(cats, "Timestamp")
	}
	return cats
}

func countLeaks(file *os.File, kind imgutil.Kind) (int, error) {
	switch kind {
	case imgutil.KindJPEG, imgutil.KindTIFF:
		analysis, err := analyzeExif(file)
		if err != nil {
			return 0, err
		}
		return analysis.GPSCount + analysis.SerialCount, nil
	case imgutil.KindPNG:
		return 0, nil
	default:
		return 0, nil
	}
}

func cleanFile(file *os.File, job Job, kind imgutil.Kind, opts Options) (int64, error) {
	if kind == imgutil.KindTIFF {
		return 0, fmt.Errorf("TIFF stripping not implemented")
	}

	srcInfo, err := file.Stat()
	if err != nil {
		return 0, err
	}

	destPath, destDir, err := resolveDestination(job, opts)
	if err != nil {
		return 0, err
	}

	if err := os.MkdirAll(destDir, 0o755); err != nil {
		return 0, err
	}

	tmpFile, err := os.CreateTemp(destDir, "bleach-*.tmp")
	if err != nil {
		return 0, err
	}
	defer os.Remove(tmpFile.Name())

	if err := tmpFile.Chmod(srcInfo.Mode()); err != nil {
		_ = tmpFile.Close()
		return 0, err
	}

	var stripErr error
	switch kind {
	case imgutil.KindJPEG:
		stripErr = stripJPEG(file, tmpFile, opts.PreserveICC)
	case imgutil.KindPNG:
		stripErr = stripPNG(file, tmpFile, opts.PreserveICC)
	default:
		stripErr = fmt.Errorf("unsupported type")
	}

	if stripErr != nil {
		_ = tmpFile.Close()
		return 0, stripErr
	}

	if err := tmpFile.Sync(); err != nil {
		_ = tmpFile.Close()
		return 0, err
	}
	if err := tmpFile.Close(); err != nil {
		return 0, err
	}

	if err := replaceFile(tmpFile.Name(), destPath); err != nil {
		return 0, err
	}

	outInfo, err := os.Stat(destPath)
	if err != nil {
		return 0, err
	}

	return srcInfo.Size() - outInfo.Size(), nil
}

func resolveDestination(job Job, opts Options) (string, string, error) {
	if opts.InPlace {
		destDir := filepath.Dir(job.Path)
		return job.Path, destDir, nil
	}
	if opts.OutputDir == "" {
		return "", "", fmt.Errorf("output directory required when not using --inplace")
	}

	destPath := filepath.Join(opts.OutputDir, job.RelPath)
	if filepath.Clean(destPath) == filepath.Clean(job.Path) {
		return "", "", fmt.Errorf("output path resolves to input path; use --inplace or a different --output")
	}

	return destPath, filepath.Dir(destPath), nil
}

func replaceFile(tmpPath, destPath string) error {
	if err := os.Rename(tmpPath, destPath); err == nil {
		return nil
	}
	if err := os.Remove(destPath); err != nil && !os.IsNotExist(err) {
		return err
	}
	return os.Rename(tmpPath, destPath)
}

func isWithin(path string, root string) bool {
	rel, err := filepath.Rel(root, path)
	if err != nil {
		return false
	}
	if rel == "." {
		return true
	}
	if strings.HasPrefix(rel, "..") || strings.HasPrefix(rel, "..\\") || strings.HasPrefix(rel, "../") {
		return false
	}
	return true
}
