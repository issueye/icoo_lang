package core

import (
	"fmt"
	"strings"
	"time"

	"icoo_lang/internal/runtime"
	"icoo_lang/internal/stdlib/utils"
)

const defaultTimeLayout = "YYYY-MM-DDTHH:mm:ss.SSSZ"

func LoadStdTimeModule() *runtime.Module {
	return &runtime.Module{
		Name: "std.time",
		Path: "std.time",
		Exports: map[string]runtime.Value{
			"add":      &runtime.NativeFunction{Name: "add", Arity: 2, Fn: timeAdd},
			"diff":     &runtime.NativeFunction{Name: "diff", Arity: 2, Fn: timeDiff},
			"format":   &runtime.NativeFunction{Name: "format", Arity: -1, Fn: timeFormat},
			"fromUnix": &runtime.NativeFunction{Name: "fromUnix", Arity: 1, Fn: timeFromUnix},
			"now":      &runtime.NativeFunction{Name: "now", Arity: 0, Fn: timeNow},
			"parse":    &runtime.NativeFunction{Name: "parse", Arity: -1, Fn: timeParse},
			"parts":    &runtime.NativeFunction{Name: "parts", Arity: -1, Fn: timeParts},
			"sleep":    &runtime.NativeFunction{Name: "sleep", Arity: 1, Fn: timeSleep},
			"unix":     &runtime.NativeFunction{Name: "unix", Arity: 1, Fn: timeUnix},
		},
		Done: true,
	}
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
