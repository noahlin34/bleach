package processor

import (
	"bytes"
	"errors"
	"io"
	"strings"

	exif "github.com/dsoprea/go-exif/v3"
)

type ExifAnalysis struct {
	HasGPS       bool
	GPSCount     int
	HasModel     bool
	HasTimestamp bool
	SerialCount  int
}

func analyzeExif(rs io.ReadSeeker) (ExifAnalysis, error) {
	analysis := ExifAnalysis{}

	if _, err := rs.Seek(0, io.SeekStart); err != nil {
		return analysis, err
	}

	header := make([]byte, 4)
	if _, err := io.ReadFull(rs, header); err != nil {
		return analysis, err
	}
	if _, err := rs.Seek(0, io.SeekStart); err != nil {
		return analysis, err
	}

	var tags []exif.ExifTag
	var err error
	if isTIFFHeader(header) {
		tags, _, err = exif.GetFlatExifDataUniversalSearchWithReadSeeker(rs, nil, true)
	} else {
		rawExif, exifErr := exif.SearchAndExtractExifWithReader(rs)
		if exifErr != nil {
			if errorsIsNoExif(exifErr) {
				return analysis, nil
			}
			return analysis, exifErr
		}
		tags, _, err = exif.GetFlatExifDataUniversalSearchWithReadSeeker(bytes.NewReader(rawExif), nil, true)
	}
	if err != nil {
		if errorsIsNoExif(err) {
			return analysis, nil
		}
		return analysis, err
	}

	for _, tag := range tags {
		name := tag.TagName
		lower := strings.ToLower(name)

		if strings.HasPrefix(name, "GPS") || strings.Contains(tag.IfdPath, "GPS") {
			analysis.HasGPS = true
			analysis.GPSCount++
		}
		if name == "Model" || name == "CameraModelName" {
			analysis.HasModel = true
		}
		if name == "DateTimeOriginal" || name == "DateTimeDigitized" || name == "DateTime" {
			analysis.HasTimestamp = true
		}
		if strings.Contains(lower, "serial") {
			analysis.SerialCount++
		}
	}

	return analysis, nil
}

func errorsIsNoExif(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, exif.ErrNoExif) {
		return true
	}
	return strings.Contains(strings.ToLower(err.Error()), "no exif")
}

func isTIFFHeader(header []byte) bool {
	if len(header) < 4 {
		return false
	}
	if header[0] == 0x49 && header[1] == 0x49 && header[2] == 0x2a && header[3] == 0x00 {
		return true
	}
	if header[0] == 0x4d && header[1] == 0x4d && header[2] == 0x00 && header[3] == 0x2a {
		return true
	}
	return false
}
