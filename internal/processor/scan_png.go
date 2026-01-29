package processor

import (
	"bufio"
	"bytes"
	"compress/zlib"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"strings"
)

type PngAnalysis struct {
	HasGPS          bool
	HasModel        bool
	HasTimestamp    bool
	GPSValues       []string
	ModelValues     []string
	TimestampValues []string
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
			key, value := extractPNGText(chunkName, data)
			if key != "" {
				applyKeyToPngAnalysis(&analysis, key, value)
			}
		case "tIME":
			analysis.HasTimestamp = true
			if length == 7 {
				timeData := make([]byte, 7)
				if _, err := io.ReadFull(br, timeData); err != nil {
					return analysis, err
				}
				if _, err := io.CopyN(io.Discard, br, 4); err != nil {
					return analysis, err
				}
				ts := formatPNGTime(timeData)
				if ts != "" {
					analysis.TimestampValues = appendUnique(analysis.TimestampValues, "tIME="+ts)
				}
			} else {
				if _, err := io.CopyN(io.Discard, br, int64(length)+4); err != nil {
					return analysis, err
				}
			}
		case "eXIf":
			exifData := make([]byte, length)
			if _, err := io.ReadFull(br, exifData); err != nil {
				return analysis, err
			}
			if _, err := io.CopyN(io.Discard, br, 4); err != nil {
				return analysis, err
			}
			exifAnalysis, err := analyzeExif(bytes.NewReader(exifData))
			if err != nil {
				return analysis, err
			}
			mergeExifIntoPNG(&analysis, exifAnalysis)
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

func extractPNGText(chunkType string, data []byte) (string, string) {
	switch chunkType {
	case "tEXt":
		return parsePNGText(data)
	case "zTXt":
		return parsePNGZTxt(data)
	case "iTXt":
		return parsePNGiTxt(data)
	default:
		return "", ""
	}
}

func parsePNGText(data []byte) (string, string) {
	idx := indexByte(data, 0)
	if idx <= 0 {
		return "", ""
	}
	key := string(data[:idx])
	value := string(data[idx+1:])
	return key, sanitizeValue(value)
}

func parsePNGZTxt(data []byte) (string, string) {
	idx := indexByte(data, 0)
	if idx <= 0 || idx+1 >= len(data) {
		return "", ""
	}
	key := string(data[:idx])
	method := data[idx+1]
	if method != 0 {
		return key, "compressed"
	}
	compressed := data[idx+2:]
	reader, err := zlib.NewReader(bytes.NewReader(compressed))
	if err != nil {
		return key, "compressed"
	}
	defer reader.Close()
	decoded, err := io.ReadAll(reader)
	if err != nil {
		return key, "compressed"
	}
	return key, sanitizeValue(string(decoded))
}

func parsePNGiTxt(data []byte) (string, string) {
	idx := indexByte(data, 0)
	if idx <= 0 || idx+2 >= len(data) {
		return "", ""
	}
	key := string(data[:idx])
	compressionFlag := data[idx+1]
	compressionMethod := data[idx+2]
	if compressionFlag != 0 && compressionMethod != 0 {
		return key, "compressed"
	}
	rest := data[idx+3:]
	langEnd := indexByte(rest, 0)
	if langEnd < 0 {
		return key, ""
	}
	rest = rest[langEnd+1:]
	transEnd := indexByte(rest, 0)
	if transEnd < 0 {
		return key, ""
	}
	textBytes := rest[transEnd+1:]
	if compressionFlag == 1 {
		reader, err := zlib.NewReader(bytes.NewReader(textBytes))
		if err != nil {
			return key, "compressed"
		}
		defer reader.Close()
		decoded, err := io.ReadAll(reader)
		if err != nil {
			return key, "compressed"
		}
		return key, sanitizeValue(string(decoded))
	}
	return key, sanitizeValue(string(textBytes))
}

func applyKeyToPngAnalysis(analysis *PngAnalysis, key string, value string) {
	lower := strings.ToLower(key)
	entry := fmtKeyValue(key, value)
	if strings.Contains(lower, "gps") || strings.Contains(lower, "latitude") || strings.Contains(lower, "longitude") {
		analysis.HasGPS = true
		analysis.GPSValues = appendUnique(analysis.GPSValues, entry)
	}
	if strings.Contains(lower, "model") || strings.Contains(lower, "make") {
		analysis.HasModel = true
		analysis.ModelValues = appendUnique(analysis.ModelValues, entry)
	}
	if strings.Contains(lower, "date") || strings.Contains(lower, "time") {
		analysis.HasTimestamp = true
		analysis.TimestampValues = appendUnique(analysis.TimestampValues, entry)
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

func formatPNGTime(data []byte) string {
	if len(data) != 7 {
		return ""
	}
	year := binary.BigEndian.Uint16(data[0:2])
	month := data[2]
	day := data[3]
	hour := data[4]
	minute := data[5]
	second := data[6]
	return fmt.Sprintf("%04d-%02d-%02d %02d:%02d:%02d", year, month, day, hour, minute, second)
}

func mergeExifIntoPNG(analysis *PngAnalysis, exifAnalysis ExifAnalysis) {
	if exifAnalysis.HasGPS {
		analysis.HasGPS = true
	}
	if exifAnalysis.HasModel {
		analysis.HasModel = true
	}
	if exifAnalysis.HasTimestamp {
		analysis.HasTimestamp = true
	}
	analysis.GPSValues = appendUniqueSlice(analysis.GPSValues, exifAnalysis.GPSValues)
	analysis.ModelValues = appendUniqueSlice(analysis.ModelValues, exifAnalysis.ModelValues)
	analysis.TimestampValues = appendUniqueSlice(analysis.TimestampValues, exifAnalysis.TimestampValues)
}

func appendUniqueSlice(values []string, incoming []string) []string {
	for _, value := range incoming {
		values = appendUnique(values, value)
	}
	return values
}
