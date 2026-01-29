package imgutil

import (
	"errors"
	"io"
	"os"
)

// Kind identifies a supported image type.
type Kind int

const (
	KindUnknown Kind = iota
	KindJPEG
	KindPNG
	KindTIFF
)

func (k Kind) String() string {
	switch k {
	case KindJPEG:
		return "jpeg"
	case KindPNG:
		return "png"
	case KindTIFF:
		return "tiff"
	default:
		return "unknown"
	}
}

var (
	pngSig    = []byte{0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a}
	jpegSig   = []byte{0xff, 0xd8, 0xff}
	tiffSigLE = []byte{0x49, 0x49, 0x2a, 0x00}
	tiffSigBE = []byte{0x4d, 0x4d, 0x00, 0x2a}
)

// DetectHeader inspects the first 8 bytes of a file for known signatures.
func DetectHeader(header []byte) (Kind, error) {
	if len(header) < 8 {
		return KindUnknown, errors.New("header too short")
	}

	if hasPrefix(header, jpegSig) {
		return KindJPEG, nil
	}
	if hasPrefix(header, pngSig) {
		return KindPNG, nil
	}
	if hasPrefix(header, tiffSigLE) || hasPrefix(header, tiffSigBE) {
		return KindTIFF, nil
	}

	return KindUnknown, nil
}

// SniffFile reads the first 8 bytes of a file to determine its type.
func SniffFile(path string) (Kind, error) {
	f, err := os.Open(path)
	if err != nil {
		return KindUnknown, err
	}
	defer f.Close()

	return SniffReader(f)
}

// SniffReader reads the first 8 bytes from r and determines its type.
func SniffReader(r io.Reader) (Kind, error) {
	header := make([]byte, 8)
	if _, err := io.ReadFull(r, header); err != nil {
		return KindUnknown, err
	}

	return DetectHeader(header)
}

func hasPrefix(buf, prefix []byte) bool {
	if len(buf) < len(prefix) {
		return false
	}
	for i := range prefix {
		if buf[i] != prefix[i] {
			return false
		}
	}
	return true
}
