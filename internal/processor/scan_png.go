package processor

import (
	"bufio"
	"encoding/binary"
	"errors"
	"io"
	"strings"
)

type PngAnalysis struct {
	HasGPS       bool
	HasModel     bool
	HasTimestamp bool
}

func scanPNGMetadata(rs io.ReadSeeker) (PngAnalysis, error) {
	analysis := PngAnalysis{}

	if _, err := rs.Seek(0, io.SeekStart); err != nil {
		return analysis, err
	}

	br := bufio.NewReader(rs)

	sig := make([]byte, 8)
	if _, err := io.ReadFull(br, sig); err != nil {
		return analysis, err
	}
	if !bytesEqual(sig, pngSignature) {
		return analysis, errors.New("invalid PNG signature")
	}

	for {
		lenBuf := make([]byte, 4)
		if _, err := io.ReadFull(br, lenBuf); err != nil {
			if err == io.EOF {
				return analysis, nil
			}
			return analysis, err
		}
		length := binary.BigEndian.Uint32(lenBuf)

		chunkType := make([]byte, 4)
		if _, err := io.ReadFull(br, chunkType); err != nil {
			return analysis, err
		}

		chunkName := string(chunkType)

		switch chunkName {
		case "tEXt", "zTXt", "iTXt":
			data := make([]byte, length)
			if _, err := io.ReadFull(br, data); err != nil {
				return analysis, err
			}
			if _, err := io.CopyN(io.Discard, br, 4); err != nil {
				return analysis, err
			}
			key := extractPNGTextKey(data)
			if key != "" {
				applyKeyToPngAnalysis(&analysis, key)
			}
		case "tIME":
			analysis.HasTimestamp = true
			if _, err := io.CopyN(io.Discard, br, int64(length)+4); err != nil {
				return analysis, err
			}
		default:
			if _, err := io.CopyN(io.Discard, br, int64(length)+4); err != nil {
				return analysis, err
			}
		}

		if chunkName == "IEND" {
			return analysis, nil
		}
	}
}

func extractPNGTextKey(data []byte) string {
	idx := indexByte(data, 0)
	if idx <= 0 {
		return ""
	}
	key := string(data[:idx])
	return key
}

func applyKeyToPngAnalysis(analysis *PngAnalysis, key string) {
	lower := strings.ToLower(key)
	if strings.Contains(lower, "gps") || strings.Contains(lower, "latitude") || strings.Contains(lower, "longitude") {
		analysis.HasGPS = true
	}
	if strings.Contains(lower, "model") || strings.Contains(lower, "make") {
		analysis.HasModel = true
	}
	if strings.Contains(lower, "date") || strings.Contains(lower, "time") {
		analysis.HasTimestamp = true
	}
}

func indexByte(data []byte, b byte) int {
	for i, v := range data {
		if v == b {
			return i
		}
	}
	return -1
}
