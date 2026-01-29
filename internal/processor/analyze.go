package processor

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"strings"

	exif "github.com/dsoprea/go-exif/v3"
	exifcommon "github.com/dsoprea/go-exif/v3/common"
)

type ExifAnalysis struct {
	HasGPS          bool
	GPSCount        int
	HasModel        bool
	HasTimestamp    bool
	SerialCount     int
	GPSValues       []string
	ModelValues     []string
	TimestampValues []string
	SerialValues    []string
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
		value := exifValueString(tag)

		if strings.HasPrefix(name, "GPS") || strings.Contains(tag.IfdPath, "GPS") {
			analysis.HasGPS = true
			analysis.GPSCount++
			if value != "" {
				analysis.GPSValues = appendUnique(analysis.GPSValues, fmtKeyValue(name, value))
			}
		}
		if name == "Model" || name == "CameraModelName" || name == "Make" {
			analysis.HasModel = true
			if value != "" {
				analysis.ModelValues = appendUnique(analysis.ModelValues, fmtKeyValue(name, value))
			}
		}
		if name == "DateTimeOriginal" || name == "DateTimeDigitized" || name == "DateTime" {
			analysis.HasTimestamp = true
			if value != "" {
				analysis.TimestampValues = appendUnique(analysis.TimestampValues, fmtKeyValue(name, value))
			}
		}
		if strings.Contains(lower, "serial") {
			analysis.SerialCount++
			if value != "" {
				analysis.SerialValues = appendUnique(analysis.SerialValues, fmtKeyValue(name, value))
			}
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

func exifValueString(tag exif.ExifTag) string {
	if tag.Value == nil {
		return ""
	}

	switch v := tag.Value.(type) {
	case string:
		return strings.TrimRight(v, "\x00")
	default:
		if formatted, err := exifcommon.FormatFromType(tag.Value, false); err == nil {
			return strings.TrimSpace(formatted)
		}
	}

	return strings.TrimSpace(fmt.Sprint(tag.Value))
}

func appendUnique(values []string, value string) []string {
	if value == "" {
		return values
	}
	for _, existing := range values {
		if existing == value {
			return values
		}
	}
	return append(values, value)
}

func fmtKeyValue(key string, value string) string {
	value = sanitizeValue(value)
	if key == "" {
		return value
	}
	return key + "=" + value
}

func sanitizeValue(value string) string {
	value = strings.ReplaceAll(value, "\r", " ")
	value = strings.ReplaceAll(value, "\n", " ")
	return strings.TrimSpace(value)
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
