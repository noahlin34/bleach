package processor

import (
	"bytes"
	"encoding/binary"
	"hash/crc32"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"testing"

	"bleach/pkg/imgutil"
)

func TestScanCleanJPEG(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "sample.jpg")

	if err := buildJPEGWithExif(src); err != nil {
		t.Fatalf("build JPEG: %v", err)
	}

	details := scanDetails(t, src, imgutil.KindJPEG)
	if !hasDetail(details, "Device Model") || !hasDetail(details, "Timestamp") {
		t.Fatalf("expected model and timestamp details, got: %#v", details)
	}

	if err := cleanToOutput(t, src, filepath.Join(dir, "out"), imgutil.KindJPEG); err != nil {
		t.Fatalf("clean JPEG: %v", err)
	}

	cleaned := filepath.Join(dir, "out", "sample.jpg")
	cleanDetails := scanDetails(t, cleaned, imgutil.KindJPEG)
	if len(cleanDetails) != 0 {
		t.Fatalf("expected no details after clean, got: %#v", cleanDetails)
	}
}

func TestScanCleanPNG(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "sample.png")

	if err := buildPNGWithMetadata(src); err != nil {
		t.Fatalf("build PNG: %v", err)
	}

	details := scanDetails(t, src, imgutil.KindPNG)
	if !hasDetail(details, "Device Model") || !hasDetail(details, "Timestamp") {
		t.Fatalf("expected model and timestamp details, got: %#v", details)
	}

	if err := cleanToOutput(t, src, filepath.Join(dir, "out"), imgutil.KindPNG); err != nil {
		t.Fatalf("clean PNG: %v", err)
	}

	cleaned := filepath.Join(dir, "out", "sample.png")
	cleanDetails := scanDetails(t, cleaned, imgutil.KindPNG)
	if len(cleanDetails) != 0 {
		t.Fatalf("expected no details after clean, got: %#v", cleanDetails)
	}
}

func scanDetails(t *testing.T, path string, kind imgutil.Kind) []ScanDetail {
	t.Helper()

	file, err := os.Open(path)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer file.Close()

	details, err := scanFile(file, kind)
	if err != nil {
		t.Fatalf("scan: %v", err)
	}
	return details
}

func hasDetail(details []ScanDetail, category string) bool {
	for _, detail := range details {
		if detail.Category == category && len(detail.Values) > 0 {
			return true
		}
	}
	return false
}

func cleanToOutput(t *testing.T, srcPath, outDir string, kind imgutil.Kind) error {
	t.Helper()

	file, err := os.Open(srcPath)
	if err != nil {
		return err
	}
	defer file.Close()

	if _, err := file.Seek(0, 0); err != nil {
		return err
	}

	job := Job{
		Path:    srcPath,
		RelPath: filepath.Base(srcPath),
		Display: filepath.Base(srcPath),
	}

	_, err = cleanFile(file, job, kind, Options{Mode: ModeClean, OutputDir: outDir})
	return err
}

func buildJPEGWithExif(path string) error {
	exifData := buildExifTIFF()
	exif := append([]byte("Exif\x00\x00"), exifData...)

	var buf bytes.Buffer
	buf.Write([]byte{0xff, 0xd8})
	buf.Write([]byte{0xff, 0xe1})
	_ = binary.Write(&buf, binary.BigEndian, uint16(len(exif)+2))
	buf.Write(exif)
	buf.Write([]byte{0xff, 0xd9})

	return os.WriteFile(path, buf.Bytes(), 0o644)
}

func buildExifTIFF() []byte {
	var tiff bytes.Buffer
	tiff.Write([]byte{0x49, 0x49, 0x2a, 0x00})
	_ = binary.Write(&tiff, binary.LittleEndian, uint32(8))
	_ = binary.Write(&tiff, binary.LittleEndian, uint16(2))
	_ = binary.Write(&tiff, binary.LittleEndian, uint16(0x0110))
	_ = binary.Write(&tiff, binary.LittleEndian, uint16(2))
	_ = binary.Write(&tiff, binary.LittleEndian, uint32(8))
	_ = binary.Write(&tiff, binary.LittleEndian, uint32(38))
	_ = binary.Write(&tiff, binary.LittleEndian, uint16(0x0132))
	_ = binary.Write(&tiff, binary.LittleEndian, uint16(2))
	_ = binary.Write(&tiff, binary.LittleEndian, uint32(20))
	_ = binary.Write(&tiff, binary.LittleEndian, uint32(46))
	_ = binary.Write(&tiff, binary.LittleEndian, uint32(0))
	tiff.Write([]byte("TestCam\x00"))
	tiff.Write([]byte("2024:01:02 03:04:05\x00"))
	return tiff.Bytes()
}

func buildPNGWithMetadata(path string) error {
	img := image.NewRGBA(image.Rect(0, 0, 1, 1))
	img.Set(0, 0, color.RGBA{R: 0xff, A: 0xff})

	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return err
	}
	data := buf.Bytes()
	if len(data) < 12 || string(data[len(data)-8:len(data)-4]) != "IEND" {
		return os.ErrInvalid
	}

	textChunk := buildPNGChunk("tEXt", []byte("Model\x00TestCam"))
	timeChunk := buildPNGChunk("tIME", []byte{0x07, 0xE8, 0x01, 0x02, 0x03, 0x04, 0x05})
	exifChunk := buildPNGChunk("eXIf", buildExifTIFF())

	insertAt := len(data) - 12
	out := append([]byte{}, data[:insertAt]...)
	out = append(out, textChunk...)
	out = append(out, timeChunk...)
	out = append(out, exifChunk...)
	out = append(out, data[insertAt:]...)

	return os.WriteFile(path, out, 0o644)
}

func buildPNGChunk(chunkType string, data []byte) []byte {
	chunkTypeBytes := []byte(chunkType)
	length := uint32(len(data))
	lenBuf := make([]byte, 4)
	binary.BigEndian.PutUint32(lenBuf, length)
	crc := crc32.ChecksumIEEE(append(chunkTypeBytes, data...))
	crcBuf := make([]byte, 4)
	binary.BigEndian.PutUint32(crcBuf, crc)

	chunk := make([]byte, 0, 12+len(data))
	chunk = append(chunk, lenBuf...)
	chunk = append(chunk, chunkTypeBytes...)
	chunk = append(chunk, data...)
	chunk = append(chunk, crcBuf...)
	return chunk
}
