package processor

import (
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

	tags, _, err := exif.GetFlatExifDataUniversalSearchWithReadSeeker(rs, nil, true)
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
	return strings.Contains(strings.ToLower(err.Error()), "no exif")
}
