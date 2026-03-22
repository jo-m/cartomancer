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

	parseInt := func(v string) int64 {
		if v == "" {
			return 0
		}
		n, _ := strconv.ParseInt(v, 10, 64)
		return n
	}

	days := parseInt(m[1])
	hours := parseInt(m[2])
	minutes := parseInt(m[3])
	seconds := parseInt(m[4])

	return time.Duration(days)*24*time.Hour +
		time.Duration(hours)*time.Hour +
		time.Duration(minutes)*time.Minute +
		time.Duration(seconds)*time.Second, nil
}
