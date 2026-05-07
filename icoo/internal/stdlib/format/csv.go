package format

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"os"
	"sort"
	"strings"

	"icoo_lang/internal/runtime"
	"icoo_lang/internal/stdlib/utils"
)

func LoadStdCSVModule() *runtime.Module {
	return &runtime.Module{
		Name: "std.data.csv",
	Path: "std.data.csv",
		Exports: map[string]runtime.Value{
			"decode":     &runtime.NativeFunction{Name: "decode", Arity: -1, Fn: csvDecode},
			"encode":     &runtime.NativeFunction{Name: "encode", Arity: -1, Fn: csvEncode},
			"fromFile":   &runtime.NativeFunction{Name: "fromFile", Arity: -1, Fn: csvFromFile},
			"saveToFile": &runtime.NativeFunction{Name: "saveToFile", Arity: -1, Fn: csvSaveToFile},
		},
		Done: true,
	}
}

type csvOptions struct {
	delimiter        rune
	header           bool
	trimLeadingSpace bool
	headers          []string
}

func defaultCSVOptions() csvOptions {
	return csvOptions{
		delimiter: ',',
		header:    true,
	}
}

func csvDecode(args []runtime.Value) (runtime.Value, error) {
	if len(args) < 1 || len(args) > 2 {
		return nil, fmt.Errorf("decode expects text and optional options")
	}
	text, err := utils.RequireStringArg("decode", args[0])
	if err != nil {
		return nil, err
	}
	opts := defaultCSVOptions()
	if len(args) == 2 {
		parsed, err := parseCSVOptions("decode", args[1])
		if err != nil {
			return nil, err
		}
		opts = parsed
	}
	reader := csv.NewReader(strings.NewReader(text))
	reader.Comma = opts.delimiter
	reader.TrimLeadingSpace = opts.trimLeadingSpace
	records, err := reader.ReadAll()
	if err != nil {
		return nil, err
	}
	if len(records) == 0 {
		return &runtime.ArrayValue{Elements: []runtime.Value{}}, nil
	}
	if !opts.header {
		return csvRecordsToArray(records), nil
	}
	headers := records[0]
	rows := make([]runtime.Value, 0, maxInt(len(records)-1, 0))
	for _, record := range records[1:] {
		fields := make(map[string]runtime.Value, len(headers))
		for i, header := range headers {
			value := ""
			if i < len(record) {
				value = record[i]
			}
			fields[header] = runtime.StringValue{Value: value}
		}
		rows = append(rows, &runtime.ObjectValue{Fields: fields})
	}
	return &runtime.ArrayValue{Elements: rows}, nil
}

func csvEncode(args []runtime.Value) (runtime.Value, error) {
	if len(args) < 1 || len(args) > 2 {
		return nil, fmt.Errorf("encode expects rows and optional options")
	}
	rows, ok := args[0].(*runtime.ArrayValue)
	if !ok {
		return nil, fmt.Errorf("encode expects array rows")
	}
	opts := defaultCSVOptions()
	if len(args) == 2 {
		parsed, err := parseCSVOptions("encode", args[1])
		if err != nil {
			return nil, err
		}
		opts = parsed
	}

	var buf bytes.Buffer
	writer := csv.NewWriter(&buf)
	writer.Comma = opts.delimiter

	if len(rows.Elements) == 0 {
		writer.Flush()
		return runtime.StringValue{Value: buf.String()}, writer.Error()
	}

	switch rows.Elements[0].(type) {
	case *runtime.ObjectValue:
		headers, err := csvHeadersForObjects(rows, opts)
		if err != nil {
			return nil, err
		}
		if opts.header {
			if err := writer.Write(headers); err != nil {
				return nil, err
			}
		}
		for _, rowValue := range rows.Elements {
			row, ok := rowValue.(*runtime.ObjectValue)
			if !ok {
				return nil, fmt.Errorf("encode rows must be all objects")
			}
			record := make([]string, 0, len(headers))
			for _, header := range headers {
				record = append(record, csvCellString(row.Fields[header]))
			}
			if err := writer.Write(record); err != nil {
				return nil, err
			}
		}
	default:
		if opts.header && len(opts.headers) > 0 {
			if err := writer.Write(opts.headers); err != nil {
				return nil, err
			}
		}
		for _, rowValue := range rows.Elements {
			row, ok := rowValue.(*runtime.ArrayValue)
			if !ok {
				return nil, fmt.Errorf("encode rows must be all arrays")
			}
			record := make([]string, 0, len(row.Elements))
			for _, cell := range row.Elements {
				record = append(record, csvCellString(cell))
			}
			if err := writer.Write(record); err != nil {
				return nil, err
			}
		}
	}
	writer.Flush()
	if err := writer.Error(); err != nil {
		return nil, err
	}
	return runtime.StringValue{Value: buf.String()}, nil
}

func csvFromFile(args []runtime.Value) (runtime.Value, error) {
	if len(args) < 1 || len(args) > 2 {
		return nil, fmt.Errorf("fromFile expects path and optional options")
	}
	path, err := utils.RequireStringArg("fromFile", args[0])
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	callArgs := []runtime.Value{runtime.StringValue{Value: string(data)}}
	if len(args) == 2 {
		callArgs = append(callArgs, args[1])
	}
	return csvDecode(callArgs)
}

func csvSaveToFile(args []runtime.Value) (runtime.Value, error) {
	if len(args) < 2 || len(args) > 3 {
		return nil, fmt.Errorf("saveToFile expects path, rows, and optional options")
	}
	path, err := utils.RequireStringArg("saveToFile", args[0])
	if err != nil {
		return nil, err
	}
	callArgs := []runtime.Value{args[1]}
	if len(args) == 3 {
		callArgs = append(callArgs, args[2])
	}
	encoded, err := csvEncode(callArgs)
	if err != nil {
		return nil, err
	}
	text := encoded.(runtime.StringValue)
	if err := os.WriteFile(path, []byte(text.Value), 0o644); err != nil {
		return nil, err
	}
	return runtime.NullValue{}, nil
}

func parseCSVOptions(name string, value runtime.Value) (csvOptions, error) {
	options, ok := value.(*runtime.ObjectValue)
	if !ok {
		return csvOptions{}, fmt.Errorf("%s expects options object", name)
	}
	opts := defaultCSVOptions()
	if delimiterValue, ok := options.Fields["delimiter"]; ok {
		delimiter, err := utils.RequireStringArg(name, delimiterValue)
		if err != nil {
			return csvOptions{}, err
		}
		runes := []rune(delimiter)
		if len(runes) != 1 {
			return csvOptions{}, fmt.Errorf("%s delimiter must be single character", name)
		}
		opts.delimiter = runes[0]
	}
	if headerValue, ok := options.Fields["header"]; ok {
		boolValue, ok := headerValue.(runtime.BoolValue)
		if !ok {
			return csvOptions{}, fmt.Errorf("%s header must be bool", name)
		}
		opts.header = boolValue.Value
	}
	if trimValue, ok := options.Fields["trimLeadingSpace"]; ok {
		boolValue, ok := trimValue.(runtime.BoolValue)
		if !ok {
			return csvOptions{}, fmt.Errorf("%s trimLeadingSpace must be bool", name)
		}
		opts.trimLeadingSpace = boolValue.Value
	}
	if headersValue, ok := options.Fields["headers"]; ok {
		arrayValue, ok := headersValue.(*runtime.ArrayValue)
		if !ok {
			return csvOptions{}, fmt.Errorf("%s headers must be array", name)
		}
		headers := make([]string, 0, len(arrayValue.Elements))
		for _, item := range arrayValue.Elements {
			header, err := utils.RequireStringArg(name, item)
			if err != nil {
				return csvOptions{}, err
			}
			headers = append(headers, header)
		}
		opts.headers = headers
	}
	return opts, nil
}

func csvRecordsToArray(records [][]string) runtime.Value {
	rows := make([]runtime.Value, 0, len(records))
	for _, record := range records {
		items := make([]runtime.Value, 0, len(record))
		for _, cell := range record {
			items = append(items, runtime.StringValue{Value: cell})
		}
		rows = append(rows, &runtime.ArrayValue{Elements: items})
	}
	return &runtime.ArrayValue{Elements: rows}
}

func csvHeadersForObjects(rows *runtime.ArrayValue, opts csvOptions) ([]string, error) {
	if len(opts.headers) > 0 {
		return append([]string{}, opts.headers...), nil
	}
	seen := map[string]struct{}{}
	headers := []string{}
	for _, rowValue := range rows.Elements {
		row, ok := rowValue.(*runtime.ObjectValue)
		if !ok {
			return nil, fmt.Errorf("encode rows must be all objects")
		}
		for key := range row.Fields {
			if _, ok := seen[key]; ok {
				continue
			}
			seen[key] = struct{}{}
			headers = append(headers, key)
		}
	}
	sort.Strings(headers)
	return headers, nil
}

func csvCellString(value runtime.Value) string {
	if value == nil {
		return ""
	}
	switch typed := value.(type) {
	case runtime.NullValue:
		return ""
	case runtime.StringValue:
		return typed.Value
	default:
		return value.String()
	}
}

func maxInt(left, right int) int {
	if left > right {
		return left
	}
	return right
}
