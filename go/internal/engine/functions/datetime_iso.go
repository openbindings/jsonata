package functions

import (
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/openbindings/jsonata-runtime/go/internal/engine/internal/evaluator"
)

// ISO calendar interchange forms, including signed expanded years and the
// existing compact numeric offsets. This is not a parser for every ISO profile.
var isoCalendar = regexp.MustCompile(`^([+-]\d{6}|\d{4})(?:-(\d{2})(?:-(\d{2}))?)?(?:T(\d{2}):(\d{2})(?::(\d{2})(?:\.(\d+))?)?(Z|[+-]\d{2}:?\d{2})?)?$`)

func parseISOCalendar(value string) (any, error) {
	invalid := &evaluator.JSONataError{Code: "D3110", Message: "$toMillis: invalid or out-of-range ISO calendar timestamp"}
	parts := isoCalendar.FindStringSubmatch(value)
	if parts == nil || parts[1] == "-000000" {
		return nil, invalid
	}
	read := func(index, fallback int) int {
		if parts[index] == "" {
			return fallback
		}
		n, _ := strconv.Atoi(parts[index])
		return n
	}
	year, month, day := read(1, 0), read(2, 1), read(3, 1)
	hour, minute, second := read(4, 0), read(5, 0), read(6, 0)
	fraction := parts[7]
	ms, _ := strconv.Atoi((fraction + "000")[:3])
	leap := year%4 == 0 && (year%100 != 0 || year%400 == 0)
	days := [12]int{31, 28, 31, 30, 31, 30, 31, 31, 30, 31, 30, 31}
	if leap {
		days[1] = 29
	}
	if month < 1 || month > 12 || day < 1 || day > days[month-1] || hour > 24 || minute > 59 || second > 59 || (hour == 24 && (minute != 0 || second != 0 || strings.Trim(fraction, "0") != "")) {
		return nil, invalid
	}
	offset := 0
	if parts[8] != "" && parts[8] != "Z" {
		var err error
		offset, err = parseNumericTZ(parts[8])
		if err != nil {
			return nil, invalid
		}
	}
	// Every component has been checked before time.Date's deliberate calendar
	// normalization. Only valid 24:00 denotes midnight on the following day.
	instant := time.Date(year, time.Month(month), day, hour, minute, second, ms*1000000, time.UTC).UnixMilli() - int64(offset)*1000
	if instant < -8640000000000000 || instant > 8640000000000000 {
		return nil, invalid
	}
	return float64(instant), nil
}
