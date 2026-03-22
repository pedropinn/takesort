package dateextract

import (
	"regexp"
	"time"
)

// djiPattern matches DJI filenames with a 14-digit timestamp after the prefix.
// Example: DJI_20260218094732_0004_D.MP4
var djiPattern = regexp.MustCompile(`^(?i)DJI_(\d{14})_`)

// ExtractDate parses the embedded date from a DJI filename.
// It returns the parsed date and true on success, or a zero time and false
// if the filename does not match the DJI pattern or the date is invalid.
func ExtractDate(filename string) (time.Time, bool) {
	matches := djiPattern.FindStringSubmatch(filename)
	if matches == nil {
		return time.Time{}, false
	}

	dateStr := matches[1][:8]
	t, err := time.Parse("20060102", dateStr)
	if err != nil {
		return time.Time{}, false
	}

	return t, true
}
