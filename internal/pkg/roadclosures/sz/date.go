package sz

import (
	"strings"
	"time"
)

// germanMonths maps lower-cased German month names (and their common
// abbreviations) to the corresponding [time.Month]. The umlauted forms
// "März" appear in the live feed; abbreviated forms are accepted
// defensively in case the source ever switches.
var germanMonths = map[string]time.Month{
	"januar":    time.January,
	"februar":   time.February,
	"märz":      time.March,
	"maerz":     time.March,
	"april":     time.April,
	"mai":       time.May,
	"juni":      time.June,
	"juli":      time.July,
	"august":    time.August,
	"september": time.September,
	"oktober":   time.October,
	"november":  time.November,
	"dezember":  time.December,
	"jan":       time.January,
	"feb":       time.February,
	"mär":       time.March,
	"mar":       time.March,
	"apr":       time.April,
	"jun":       time.June,
	"jul":       time.July,
	"aug":       time.August,
	"sep":       time.September,
	"okt":       time.October,
	"nov":       time.November,
	"dez":       time.December,
}

// numericDateLayouts are tried in order against numeric forms of dates that
// appear in the SZ feed (e.g. "18.11.2024", "31.07.2026").
var numericDateLayouts = []string{
	"02.01.2006",
	"2.1.2006",
}

// parseGermanDate parses a date string from the SZ Baustellen feed. The feed
// mixes two formats:
//
//   - "D. MonthName YYYY"  e.g. "13. April 2026", "16. März 2026"
//   - "DD.MM.YYYY"         e.g. "18.11.2024", "31.07.2026"
//
// Values may be prefixed with "ca." or padded with surrounding whitespace.
// Returns the parsed date (UTC, midnight) and true on success; otherwise
// returns the zero time and false. Unparseable inputs are not an error;
// callers can store them as NULL.
func parseGermanDate(s string) (time.Time, bool) {
	s = strings.TrimSpace(s)
	s = strings.TrimPrefix(s, "ca.")
	s = strings.TrimPrefix(s, "ca")
	s = strings.TrimSpace(s)
	if s == "" {
		return time.Time{}, false
	}

	if t, ok := parseGermanWordDate(s); ok {
		return t, true
	}
	for _, layout := range numericDateLayouts {
		if t, err := time.Parse(layout, s); err == nil {
			return t, true
		}
	}
	return time.Time{}, false
}

// parseGermanWordDate parses dates of the form "D. MonthName YYYY", where
// MonthName is a German month name (umlauted or ASCII transliterated).
// The day component may or may not be followed by a period; whitespace
// between tokens is collapsed.
func parseGermanWordDate(s string) (time.Time, bool) {
	fields := strings.Fields(s)
	if len(fields) < 3 {
		return time.Time{}, false
	}

	dayStr := strings.TrimSuffix(fields[0], ".")
	day, err := parsePositiveInt(dayStr)
	if err != nil || day < 1 || day > 31 {
		return time.Time{}, false
	}

	month, ok := germanMonths[strings.ToLower(fields[1])]
	if !ok {
		return time.Time{}, false
	}

	year, err := parsePositiveInt(fields[2])
	if err != nil || year < 1900 || year > 2200 {
		return time.Time{}, false
	}

	return time.Date(year, month, day, 0, 0, 0, 0, time.UTC), true
}

// parsePositiveInt parses s as a non-negative decimal integer. Used by the
// German-word date parser because [strconv.Atoi] would also accept leading
// signs which would be confusing in this context.
func parsePositiveInt(s string) (int, error) {
	if s == "" {
		return 0, errEmpty
	}
	n := 0
	for _, r := range s {
		if r < '0' || r > '9' {
			return 0, errNotDigit
		}
		n = n*10 + int(r-'0')
	}
	return n, nil
}

// errEmpty is returned by parsePositiveInt for an empty input.
var errEmpty = parseErr("empty input")

// errNotDigit is returned by parsePositiveInt for a non-digit character.
var errNotDigit = parseErr("non-digit character")

// parseErr is a tiny error type so parsePositiveInt can return sentinel
// errors without pulling in fmt.
type parseErr string

// Error implements the error interface.
func (e parseErr) Error() string { return string(e) }
