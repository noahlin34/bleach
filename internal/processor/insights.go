package processor

import (
	"fmt"
	"strconv"
	"strings"

	"bleach/pkg/imgutil"
)

func buildInsights(kind imgutil.Kind, details []ScanDetail) []ScanInsight {
	if len(details) == 0 {
		return nil
	}

	values := flattenDetails(details)
	insights := []ScanInsight{}

	if gps := buildGPSInsight(values); gps != nil {
		insights = append(insights, *gps)
		insights = append(insights, ScanInsight{
			Kind:    "Location",
			Message: "Exact coordinates can reveal home, workplace, or travel patterns.",
		})
	}

	if device := buildDeviceInsight(values); device != nil {
		insights = append(insights, *device)
	}

	if ts := buildTimestampInsight(values); ts != nil {
		insights = append(insights, *ts)
		insights = append(insights, ScanInsight{
			Kind:    "Timeline",
			Message: "Capture timestamps can expose routines and time zones.",
		})
	}

	if serial := buildSerialInsight(values); serial != nil {
		insights = append(insights, *serial)
	}

	return insights
}

func flattenDetails(details []ScanDetail) map[string][]string {
	values := make(map[string][]string)
	for _, detail := range details {
		for _, entry := range detail.Values {
			key, value := splitKeyValue(entry)
			if key == "" {
				continue
			}
			values[key] = append(values[key], value)
		}
	}
	return values
}

func buildGPSInsight(values map[string][]string) *ScanInsight {
	latRaw := firstValue(values, "GPSLatitude")
	lonRaw := firstValue(values, "GPSLongitude")
	if latRaw == "" || lonRaw == "" {
		return nil
	}

	latRef := firstValue(values, "GPSLatitudeRef")
	lonRef := firstValue(values, "GPSLongitudeRef")
	lat, okLat := parseGPSCoordinate(latRaw)
	lon, okLon := parseGPSCoordinate(lonRaw)
	if !okLat || !okLon {
		return nil
	}

	if latRef == "S" {
		lat = -lat
	}
	if lonRef == "W" {
		lon = -lon
	}

	msg := fmt.Sprintf("Approx location: %.5f, %.5f", lat, lon)
	return &ScanInsight{Kind: "Location", Message: msg}
}

func buildDeviceInsight(values map[string][]string) *ScanInsight {
	make := firstValue(values, "Make")
	model := firstValue(values, "Model")
	cameraModel := firstValue(values, "CameraModelName")

	device := strings.TrimSpace(strings.Join([]string{make, model}, " "))
	if device == "" {
		device = cameraModel
	}
	if device == "" {
		return nil
	}

	deviceType := inferDeviceType(strings.ToLower(device))
	msg := fmt.Sprintf("Device: %s", device)
	if deviceType != "" {
		msg += fmt.Sprintf(" (%s)", deviceType)
	}
	return &ScanInsight{Kind: "Device", Message: msg}
}

func buildTimestampInsight(values map[string][]string) *ScanInsight {
	ts := firstValue(values, "DateTimeOriginal")
	if ts == "" {
		ts = firstValue(values, "DateTimeDigitized")
	}
	if ts == "" {
		ts = firstValue(values, "DateTime")
	}
	if ts == "" {
		return nil
	}

	formatted := replaceFirstN(ts, ":", "-", 2)
	return &ScanInsight{Kind: "Timeline", Message: fmt.Sprintf("Captured: %s (timezone unknown)", formatted)}
}

func buildSerialInsight(values map[string][]string) *ScanInsight {
	for key, vals := range values {
		if strings.Contains(strings.ToLower(key), "serial") && len(vals) > 0 {
			return &ScanInsight{Kind: "Identifier", Message: "Unique device identifiers (serial numbers) are present."}
		}
	}
	return nil
}

func splitKeyValue(entry string) (string, string) {
	parts := strings.SplitN(entry, "=", 2)
	if len(parts) != 2 {
		return "", ""
	}
	return strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1])
}

func firstValue(values map[string][]string, key string) string {
	if list, ok := values[key]; ok && len(list) > 0 {
		return list[0]
	}
	return ""
}

func parseGPSCoordinate(raw string) (float64, bool) {
	raw = strings.TrimSpace(raw)
	raw = strings.TrimPrefix(raw, "[")
	raw = strings.TrimSuffix(raw, "]")
	parts := strings.Fields(raw)
	if len(parts) == 0 {
		return 0, false
	}

	if len(parts) == 1 {
		if v, err := strconv.ParseFloat(parts[0], 64); err == nil {
			return v, true
		}
	}

	values := make([]float64, 0, len(parts))
	for _, part := range parts {
		value, ok := parseRational(part)
		if !ok {
			return 0, false
		}
		values = append(values, value)
	}

	if len(values) == 3 {
		return values[0] + values[1]/60.0 + values[2]/3600.0, true
	}
	if len(values) == 2 {
		return values[0] + values[1]/60.0, true
	}
	return values[0], true
}

func parseRational(part string) (float64, bool) {
	part = strings.TrimSpace(part)
	if part == "" {
		return 0, false
	}
	if strings.Contains(part, "/") {
		items := strings.SplitN(part, "/", 2)
		if len(items) != 2 {
			return 0, false
		}
		num, err := strconv.ParseFloat(items[0], 64)
		if err != nil {
			return 0, false
		}
		den, err := strconv.ParseFloat(items[1], 64)
		if err != nil || den == 0 {
			return 0, false
		}
		return num / den, true
	}

	value, err := strconv.ParseFloat(part, 64)
	if err != nil {
		return 0, false
	}
	return value, true
}

func inferDeviceType(device string) string {
	switch {
	case strings.Contains(device, "iphone"),
		strings.Contains(device, "pixel"),
		strings.Contains(device, "galaxy"),
		strings.Contains(device, "android"):
		return "smartphone"
	case strings.Contains(device, "ipad"),
		strings.Contains(device, "tablet"):
		return "tablet"
	case strings.Contains(device, "gopro"):
		return "action camera"
	case strings.Contains(device, "dji"):
		return "drone"
	case strings.Contains(device, "canon"),
		strings.Contains(device, "nikon"),
		strings.Contains(device, "sony"),
		strings.Contains(device, "fujifilm"),
		strings.Contains(device, "panasonic"),
		strings.Contains(device, "olympus"),
		strings.Contains(device, "leica"):
		return "camera"
	default:
		return ""
	}
}

func replaceFirstN(s, old, new string, n int) string {
	if n <= 0 || old == "" {
		return s
	}
	out := s
	for i := 0; i < n; i++ {
		if idx := strings.Index(out, old); idx >= 0 {
			out = out[:idx] + new + out[idx+len(old):]
		} else {
			break
		}
	}
	return out
}
