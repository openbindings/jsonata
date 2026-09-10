package functions

import (
	"fmt"
	"time"

	"github.com/openbindings/jsonata/go/internal/engine/internal/evaluator"
	"github.com/openbindings/jsonata/go/internal/engine/internal/numeric"
)

func fnNow(args []any, _ any, env *evaluator.Environment) (any, error) {
	params := make([]any, 1, len(args)+1)
	params[0] = float64(env.Timestamp().UnixMilli())
	return fnFromMillis(append(params, args...), nil, env)
}

func fnMillis(_ []any, _ any, env *evaluator.Environment) (any, error) {
	return float64(env.Timestamp().UnixMilli()), nil
}

func fnFromMillis(args []any, focus any, env *evaluator.Environment) (any, error) {
	if len(args) == 0 {
		// Called with no arguments - use the current context ($) if available.
		if focus != nil {
			args = []any{focus}
		} else {
			return nil, nil
		}
	}
	if args[0] == nil {
		return nil, nil
	}
	if !evaluator.IsNumeric(args[0]) {
		return nil, &evaluator.JSONataError{Code: "T0410", Message: "$fromMillis: argument must be a number"}
	}
	// Match the existing Date-compatible timestamp domain, checking the exact
	// value before TimeClip's deliberate truncation toward zero.
	if numeric.Compare(args[0], float64(-8640000000000000)) < 0 || numeric.Compare(args[0], float64(8640000000000000)) > 0 {
		return nil, &evaluator.JSONataError{Code: "D3110", Message: "$fromMillis: timestamp is outside the supported date range"}
	}
	ms, ok := numeric.TruncInt64(args[0])
	if !ok {
		return nil, &evaluator.JSONataError{Code: "D3110", Message: "$fromMillis: invalid timestamp"}
	}
	t := time.UnixMilli(ms).UTC()

	// Resolve timezone if provided (args[2]).
	tz := time.UTC
	if len(args) >= 3 && args[2] != nil {
		var err error
		tz, err = parseTZ(args[2])
		if err != nil {
			return nil, err
		}
	}
	_, offset := t.In(tz).Zone()
	localMillis := ms + int64(offset)*1000
	if localMillis < -8640000000000000 || localMillis > 8640000000000000 {
		return nil, &evaluator.JSONataError{Code: "D3110", Message: "$fromMillis: local timestamp is outside the supported date range"}
	}

	if len(args) >= 2 && args[1] != nil {
		picture, ok := args[1].(string)
		if !ok {
			return nil, &evaluator.JSONataError{Code: "T0410", Message: "$fromMillis: picture argument must be a string"}
		}
		s, err := formatWithPicture(t.In(tz), picture, env.StandardLibraryOnly())
		if err != nil {
			return nil, err
		}
		return s, nil
	}
	// No picture: use default ISO 8601 format with milliseconds and timezone offset.
	return formatDefaultISO(t.In(tz)), nil
}

func formatDefaultISO(t time.Time) string {
	_, offset := t.Zone()
	ms := t.UnixMilli() % 1000
	if ms < 0 {
		ms += 1000
	}
	year := fmt.Sprintf("%04d", t.Year())
	if t.Year() < 0 {
		year = fmt.Sprintf("-%06d", -t.Year())
	} else if t.Year() > 9999 {
		year = fmt.Sprintf("+%06d", t.Year())
	}
	base := year + t.Format("-01-02T15:04:05")
	millis := fmt.Sprintf(".%03d", ms)
	if offset == 0 {
		return base + millis + "Z"
	}
	sign := "+"
	if offset < 0 {
		sign = "-"
		offset = -offset
	}
	hours := offset / 3600
	mins := (offset % 3600) / 60
	return fmt.Sprintf("%s%s%s%02d:%02d", base, millis, sign, hours, mins)
}

func fnToMillis(args []any, _ any, env *evaluator.Environment) (any, error) {
	if len(args) == 0 || args[0] == nil {
		return nil, nil
	}
	s, ok := args[0].(string)
	if !ok {
		return nil, &evaluator.JSONataError{Code: "T0410", Message: "$toMillis: argument must be a string"}
	}

	if len(args) >= 2 && args[1] != nil {
		// Picture-based parsing — use custom XPath picture parser.
		picture, ok := args[1].(string)
		if !ok {
			return nil, &evaluator.JSONataError{Code: "T0410", Message: "$toMillis: picture argument must be a string"}
		}
		t, ok2, err2 := parseWithPicture(s, picture, env.Timestamp(), env.StandardLibraryOnly())
		if err2 != nil {
			return nil, err2
		}
		if !ok2 {
			return nil, nil
		}
		return float64(t.UnixMilli()), nil
	}
	return parseISOCalendar(s)
}
