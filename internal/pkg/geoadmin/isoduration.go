package geoadmin

import (
	"fmt"
	"regexp"
	"strconv"
	"time"
)

// durationPattern matches ISO 8601 durations of the form P[n]DT[n]H[n]M[n]S.
var durationPattern = regexp.MustCompile(`^P(?:(\d+)D)?T(?:(\d+)H)?(?:(\d+)M)?(?:(\d+)S)?$`)

// ParseISO8601Duration parses an ISO 8601 duration of the form P[n]DT[n]H[n]M[n]S
// and returns the equivalent [time.Duration].
func ParseISO8601Duration(s string) (time.Duration, error) {
	m := durationPattern.FindStringSubmatch(s)
	if m == nil {
		return 0, fmt.Errorf("unsupported ISO 8601 duration format: %q", s)
	}

	parseInt := func(v string) (int64, error) {
		if v == "" {
			return 0, nil
		}
		return strconv.ParseInt(v, 10, 64)
	}

	days, err := parseInt(m[1])
	if err != nil {
		return 0, fmt.Errorf("invalid days in ISO 8601 duration %q: %w", s, err)
	}
	hours, err := parseInt(m[2])
	if err != nil {
		return 0, fmt.Errorf("invalid hours in ISO 8601 duration %q: %w", s, err)
	}
	minutes, err := parseInt(m[3])
	if err != nil {
		return 0, fmt.Errorf("invalid minutes in ISO 8601 duration %q: %w", s, err)
	}
	seconds, err := parseInt(m[4])
	if err != nil {
		return 0, fmt.Errorf("invalid seconds in ISO 8601 duration %q: %w", s, err)
	}

	return time.Duration(days)*24*time.Hour +
		time.Duration(hours)*time.Hour +
		time.Duration(minutes)*time.Minute +
		time.Duration(seconds)*time.Second, nil
}
