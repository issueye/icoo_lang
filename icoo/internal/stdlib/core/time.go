package core

import (
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"icoo_lang/internal/runtime"
	"icoo_lang/internal/stdlib/utils"
)

const defaultTimeLayout = "YYYY-MM-DDTHH:mm:ss.SSSZ"

func LoadStdTimeBasicModule() *runtime.Module {
	return &runtime.Module{
		Name: "std.time.basic",
		Path: "std.time.basic",
		Exports: map[string]runtime.Value{
			"add":      &runtime.NativeFunction{Name: "add", Arity: 2, Fn: timeAdd},
			"diff":     &runtime.NativeFunction{Name: "diff", Arity: 2, Fn: timeDiff},
			"duration": &runtime.NativeFunction{Name: "duration", Arity: 1, Fn: timeDuration},
			"format":   &runtime.NativeFunction{Name: "format", Arity: -1, Fn: timeFormat},
			"fromUnix": &runtime.NativeFunction{Name: "fromUnix", Arity: 1, Fn: timeFromUnix},
			"interval": &runtime.NativeFunction{Name: "interval", Arity: 1, Fn: timeInterval},
			"next":     &runtime.NativeFunction{Name: "next", Arity: -1, Fn: timeNext},
			"now":      &runtime.NativeFunction{Name: "now", Arity: 0, Fn: timeNow},
			"parse":    &runtime.NativeFunction{Name: "parse", Arity: -1, Fn: timeParse},
			"parts":    &runtime.NativeFunction{Name: "parts", Arity: -1, Fn: timeParts},
			"sleep":    &runtime.NativeFunction{Name: "sleep", Arity: 1, Fn: timeSleep},
			"ticker":   &runtime.NativeFunction{Name: "ticker", Arity: 1, Fn: timeTicker},
			"unix":     &runtime.NativeFunction{Name: "unix", Arity: 1, Fn: timeUnix},
		},
		Done: true,
	}
}

type timeTickerBinding struct {
	ticker *time.Ticker
	stopCh chan struct{}
	once   sync.Once
}

func timeNow(args []runtime.Value) (runtime.Value, error) {
	return runtime.IntValue{Value: time.Now().UnixMilli()}, nil
}

func timeSleep(args []runtime.Value) (runtime.Value, error) {
	ms, err := requireTimeIntArg("sleep", args[0])
	if err != nil {
		return nil, err
	}
	if ms.Value < 0 {
		return nil, fmt.Errorf("sleep expects non-negative milliseconds")
	}
	time.Sleep(time.Duration(ms.Value) * time.Millisecond)
	return runtime.NullValue{}, nil
}

func timeAdd(args []runtime.Value) (runtime.Value, error) {
	base, err := requireTimeIntArg("add", args[0])
	if err != nil {
		return nil, err
	}
	delta, err := requireTimeIntArg("add", args[1])
	if err != nil {
		return nil, err
	}
	return runtime.IntValue{Value: base.Value + delta.Value}, nil
}

func timeDiff(args []runtime.Value) (runtime.Value, error) {
	left, err := requireTimeIntArg("diff", args[0])
	if err != nil {
		return nil, err
	}
	right, err := requireTimeIntArg("diff", args[1])
	if err != nil {
		return nil, err
	}
	return runtime.IntValue{Value: left.Value - right.Value}, nil
}

func timeDuration(args []runtime.Value) (runtime.Value, error) {
	text, err := utils.RequireStringArg("duration", args[0])
	if err != nil {
		return nil, err
	}
	duration, err := parseDurationText(text)
	if err != nil {
		return nil, err
	}
	return runtime.IntValue{Value: duration.Milliseconds()}, nil
}

func timeFormat(args []runtime.Value) (runtime.Value, error) {
	if len(args) < 1 || len(args) > 3 {
		return nil, fmt.Errorf("format expects 1 to 3 arguments")
	}
	ts, err := requireTimeIntArg("format", args[0])
	if err != nil {
		return nil, err
	}
	layout, err := timeLayoutArg("format", args, 1)
	if err != nil {
		return nil, err
	}
	loc, err := timeLocationArg("format", args, 2)
	if err != nil {
		return nil, err
	}
	return runtime.StringValue{Value: time.UnixMilli(ts.Value).In(loc).Format(layout)}, nil
}

func timeParse(args []runtime.Value) (runtime.Value, error) {
	if len(args) < 1 || len(args) > 3 {
		return nil, fmt.Errorf("parse expects 1 to 3 arguments")
	}
	text, err := utils.RequireStringArg("parse", args[0])
	if err != nil {
		return nil, err
	}
	loc, err := timeLocationArg("parse", args, 2)
	if err != nil {
		return nil, err
	}
	if len(args) >= 2 {
		layoutValue, ok := args[1].(runtime.StringValue)
		if !ok {
			return nil, fmt.Errorf("parse expects string layout")
		}
		layout := translateTimeLayout(layoutValue.Value)
		parsed, err := time.ParseInLocation(layout, text, loc)
		if err != nil {
			return nil, err
		}
		return runtime.IntValue{Value: parsed.UnixMilli()}, nil
	}
	parsed, err := parseAutoTime(text, loc)
	if err != nil {
		return nil, err
	}
	return runtime.IntValue{Value: parsed.UnixMilli()}, nil
}

func timeParts(args []runtime.Value) (runtime.Value, error) {
	if len(args) < 1 || len(args) > 2 {
		return nil, fmt.Errorf("parts expects 1 or 2 arguments")
	}
	ts, err := requireTimeIntArg("parts", args[0])
	if err != nil {
		return nil, err
	}
	loc, err := timeLocationArg("parts", args, 1)
	if err != nil {
		return nil, err
	}
	current := time.UnixMilli(ts.Value).In(loc)
	_, offsetSeconds := current.Zone()
	return &runtime.ObjectValue{Fields: map[string]runtime.Value{
		"year":          runtime.IntValue{Value: int64(current.Year())},
		"month":         runtime.IntValue{Value: int64(current.Month())},
		"day":           runtime.IntValue{Value: int64(current.Day())},
		"hour":          runtime.IntValue{Value: int64(current.Hour())},
		"minute":        runtime.IntValue{Value: int64(current.Minute())},
		"second":        runtime.IntValue{Value: int64(current.Second())},
		"millisecond":   runtime.IntValue{Value: int64(current.Nanosecond() / int(time.Millisecond))},
		"weekday":       runtime.IntValue{Value: int64(current.Weekday())},
		"weekdayName":   runtime.StringValue{Value: current.Weekday().String()},
		"yearDay":       runtime.IntValue{Value: int64(current.YearDay())},
		"unix":          runtime.IntValue{Value: current.Unix()},
		"unixMilli":     runtime.IntValue{Value: current.UnixMilli()},
		"timezone":      runtime.StringValue{Value: current.Location().String()},
		"offsetSeconds": runtime.IntValue{Value: int64(offsetSeconds)},
	}}, nil
}

func timeUnix(args []runtime.Value) (runtime.Value, error) {
	ts, err := requireTimeIntArg("unix", args[0])
	if err != nil {
		return nil, err
	}
	return runtime.IntValue{Value: ts.Value / 1000}, nil
}

func timeFromUnix(args []runtime.Value) (runtime.Value, error) {
	seconds, err := requireTimeIntArg("fromUnix", args[0])
	if err != nil {
		return nil, err
	}
	return runtime.IntValue{Value: seconds.Value * 1000}, nil
}

func timeTicker(args []runtime.Value) (runtime.Value, error) {
	ms, err := requireTimeIntArg("ticker", args[0])
	if err != nil {
		return nil, err
	}
	if ms.Value <= 0 {
		return nil, fmt.Errorf("ticker expects positive interval")
	}
	binding := &timeTickerBinding{
		ticker: time.NewTicker(time.Duration(ms.Value) * time.Millisecond),
		stopCh: make(chan struct{}),
	}
	return binding.object(), nil
}

func timeInterval(args []runtime.Value) (runtime.Value, error) {
	return timeTicker(args)
}

func timeNext(args []runtime.Value) (runtime.Value, error) {
	if len(args) < 1 || len(args) > 3 {
		return nil, fmt.Errorf("next expects expression and optional from/timezone")
	}
	spec, err := utils.RequireStringArg("next", args[0])
	if err != nil {
		return nil, err
	}
	from := time.Now()
	if len(args) >= 2 {
		ts, err := requireTimeIntArg("next", args[1])
		if err != nil {
			return nil, err
		}
		from = time.UnixMilli(ts.Value)
	}
	loc, err := timeLocationArg("next", args, 2)
	if err != nil {
		return nil, err
	}
	next, err := nextCronTime(spec, from.In(loc))
	if err != nil {
		return nil, err
	}
	return runtime.IntValue{Value: next.UnixMilli()}, nil
}

func (binding *timeTickerBinding) object() *runtime.ObjectValue {
	return &runtime.ObjectValue{Fields: map[string]runtime.Value{
		"next": &runtime.NativeFunction{Name: "time.ticker.next", Arity: 0, Fn: binding.next},
		"stop": &runtime.NativeFunction{Name: "time.ticker.stop", Arity: 0, Fn: binding.stop},
	}}
}

func (binding *timeTickerBinding) next(args []runtime.Value) (runtime.Value, error) {
	select {
	case tick := <-binding.ticker.C:
		return runtime.IntValue{Value: tick.UnixMilli()}, nil
	case <-binding.stopCh:
		return runtime.NullValue{}, nil
	}
}

func (binding *timeTickerBinding) stop(args []runtime.Value) (runtime.Value, error) {
	binding.once.Do(func() {
		close(binding.stopCh)
		binding.ticker.Stop()
	})
	return runtime.NullValue{}, nil
}

func requireTimeIntArg(name string, value runtime.Value) (runtime.IntValue, error) {
	intValue, ok := value.(runtime.IntValue)
	if !ok {
		return runtime.IntValue{}, fmt.Errorf("%s expects int argument", name)
	}
	return intValue, nil
}

func timeLayoutArg(name string, args []runtime.Value, index int) (string, error) {
	if len(args) <= index {
		return translateTimeLayout(defaultTimeLayout), nil
	}
	layoutValue, ok := args[index].(runtime.StringValue)
	if !ok {
		return "", fmt.Errorf("%s expects string layout", name)
	}
	return translateTimeLayout(layoutValue.Value), nil
}

func timeLocationArg(name string, args []runtime.Value, index int) (*time.Location, error) {
	if len(args) <= index {
		return time.Local, nil
	}
	locationValue, ok := args[index].(runtime.StringValue)
	if !ok {
		return nil, fmt.Errorf("%s expects string timezone", name)
	}
	locationName := strings.TrimSpace(locationValue.Value)
	switch strings.ToUpper(locationName) {
	case "", "LOCAL":
		return time.Local, nil
	case "UTC":
		return time.UTC, nil
	default:
		return time.LoadLocation(locationName)
	}
}

func parseAutoTime(text string, loc *time.Location) (time.Time, error) {
	layouts := []string{
		time.RFC3339Nano,
		time.RFC3339,
		translateTimeLayout(defaultTimeLayout),
		"2006-01-02 15:04:05",
		"2006-01-02 15:04",
		"2006-01-02",
		"2006/01/02 15:04:05",
		"2006/01/02",
	}
	var lastErr error
	for _, layout := range layouts {
		parsed, err := time.ParseInLocation(layout, text, loc)
		if err == nil {
			return parsed, nil
		}
		lastErr = err
	}
	if lastErr == nil {
		lastErr = fmt.Errorf("unsupported time format")
	}
	return time.Time{}, lastErr
}

func parseDurationText(text string) (time.Duration, error) {
	text = strings.TrimSpace(text)
	if text == "" {
		return 0, fmt.Errorf("duration expects non-empty string")
	}
	if duration, err := time.ParseDuration(text); err == nil {
		return duration, nil
	}
	switch {
	case strings.HasSuffix(text, "d"):
		value, err := strconv.ParseFloat(strings.TrimSuffix(text, "d"), 64)
		if err != nil {
			return 0, err
		}
		return time.Duration(value * float64(24*time.Hour)), nil
	case strings.HasSuffix(text, "w"):
		value, err := strconv.ParseFloat(strings.TrimSuffix(text, "w"), 64)
		if err != nil {
			return 0, err
		}
		return time.Duration(value * float64(7*24*time.Hour)), nil
	default:
		return 0, fmt.Errorf("unsupported duration %q", text)
	}
}

type cronField struct {
	wildcard bool
	allowed  map[int]struct{}
}

func nextCronTime(spec string, from time.Time) (time.Time, error) {
	fields := strings.Fields(strings.TrimSpace(spec))
	if len(fields) != 5 {
		return time.Time{}, fmt.Errorf("next expects 5-field cron expression")
	}
	minutes, err := parseCronField(fields[0], 0, 59, false)
	if err != nil {
		return time.Time{}, err
	}
	hours, err := parseCronField(fields[1], 0, 23, false)
	if err != nil {
		return time.Time{}, err
	}
	days, err := parseCronField(fields[2], 1, 31, false)
	if err != nil {
		return time.Time{}, err
	}
	months, err := parseCronField(fields[3], 1, 12, false)
	if err != nil {
		return time.Time{}, err
	}
	weekdays, err := parseCronField(fields[4], 0, 6, true)
	if err != nil {
		return time.Time{}, err
	}

	current := from.Truncate(time.Minute).Add(time.Minute)
	limit := current.AddDate(5, 0, 0)
	for !current.After(limit) {
		if !cronMatches(months, int(current.Month())) ||
			!cronMatches(hours, current.Hour()) ||
			!cronMatches(minutes, current.Minute()) {
			current = current.Add(time.Minute)
			continue
		}
		dayMatch := cronMatches(days, current.Day())
		weekdayMatch := cronMatches(weekdays, int(current.Weekday()))
		switch {
		case days.wildcard && weekdays.wildcard:
			return current, nil
		case days.wildcard:
			if weekdayMatch {
				return current, nil
			}
		case weekdays.wildcard:
			if dayMatch {
				return current, nil
			}
		case dayMatch || weekdayMatch:
			return current, nil
		}
		current = current.Add(time.Minute)
	}
	return time.Time{}, fmt.Errorf("no matching time found within search window")
}

func parseCronField(expr string, min int, max int, sundayWrap bool) (cronField, error) {
	field := cronField{
		wildcard: expr == "*",
		allowed:  map[int]struct{}{},
	}
	if expr == "*" {
		return field, nil
	}
	for _, part := range strings.Split(expr, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			return cronField{}, fmt.Errorf("invalid cron field %q", expr)
		}
		step := 1
		rangeExpr := part
		if strings.Contains(part, "/") {
			chunks := strings.Split(part, "/")
			if len(chunks) != 2 {
				return cronField{}, fmt.Errorf("invalid cron step %q", part)
			}
			rangeExpr = chunks[0]
			parsedStep, err := strconv.Atoi(chunks[1])
			if err != nil || parsedStep <= 0 {
				return cronField{}, fmt.Errorf("invalid cron step %q", part)
			}
			step = parsedStep
		}
		start := min
		end := max
		if rangeExpr != "*" {
			if strings.Contains(rangeExpr, "-") {
				chunks := strings.Split(rangeExpr, "-")
				if len(chunks) != 2 {
					return cronField{}, fmt.Errorf("invalid cron range %q", part)
				}
				var err error
				start, err = parseCronNumber(chunks[0], sundayWrap)
				if err != nil {
					return cronField{}, err
				}
				end, err = parseCronNumber(chunks[1], sundayWrap)
				if err != nil {
					return cronField{}, err
				}
			} else {
				value, err := parseCronNumber(rangeExpr, sundayWrap)
				if err != nil {
					return cronField{}, err
				}
				start = value
				end = value
			}
		}
		if start < min || end > max || start > end {
			return cronField{}, fmt.Errorf("cron field %q is out of range", part)
		}
		for value := start; value <= end; value += step {
			field.allowed[value] = struct{}{}
		}
	}
	return field, nil
}

func parseCronNumber(text string, sundayWrap bool) (int, error) {
	value, err := strconv.Atoi(strings.TrimSpace(text))
	if err != nil {
		return 0, err
	}
	if sundayWrap && value == 7 {
		return 0, nil
	}
	return value, nil
}

func cronMatches(field cronField, value int) bool {
	if field.wildcard {
		return true
	}
	_, ok := field.allowed[value]
	return ok
}

func translateTimeLayout(layout string) string {
	if strings.TrimSpace(layout) == "" {
		return translateTimeLayout(defaultTimeLayout)
	}
	type tokenMapping struct {
		token       string
		placeholder string
		layout      string
	}
	mappings := []tokenMapping{
		{token: "YYYY", placeholder: "\x00YEAR4\x00", layout: "2006"},
		{token: "YY", placeholder: "\x00YEAR2\x00", layout: "06"},
		{token: "MM", placeholder: "\x00MONTH\x00", layout: "01"},
		{token: "DD", placeholder: "\x00DAY\x00", layout: "02"},
		{token: "HH", placeholder: "\x00HOUR\x00", layout: "15"},
		{token: "mm", placeholder: "\x00MIN\x00", layout: "04"},
		{token: "ss", placeholder: "\x00SEC\x00", layout: "05"},
		{token: "SSS", placeholder: "\x00MS\x00", layout: "000"},
		{token: "Z", placeholder: "\x00ZONE\x00", layout: "-07:00"},
	}
	translated := layout
	for _, mapping := range mappings {
		translated = strings.ReplaceAll(translated, mapping.token, mapping.placeholder)
	}
	for _, mapping := range mappings {
		translated = strings.ReplaceAll(translated, mapping.placeholder, mapping.layout)
	}
	return translated
}
