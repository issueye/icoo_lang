package core

import (
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/natefinch/lumberjack.v2"
	"icoo_lang/internal/runtime"
	"icoo_lang/internal/stdlib/utils"
)

// LoadStdCoreLogModule 加载 std.core.log 模块
func LoadStdCoreLogModule() *runtime.Module {
	return &runtime.Module{
		Name: "std.core.log",
		Path: "std.core.log",
		Exports: map[string]runtime.Value{
			"create":  &runtime.NativeFunction{Name: "create", Arity: -1, Fn: logCreate},
			"default": &runtime.NativeFunction{Name: "default", Arity: 0, Fn: logDefault},
			"levels":  logLevelsObject(),
			"log":     &runtime.NativeFunction{Name: "log", Arity: -1, Fn: logModuleLog},
			"debug":   &runtime.NativeFunction{Name: "debug", Arity: -1, Fn: logModuleDebug},
			"info":    &runtime.NativeFunction{Name: "info", Arity: -1, Fn: logModuleInfo},
			"warn":    &runtime.NativeFunction{Name: "warn", Arity: -1, Fn: logModuleWarn},
			"error":   &runtime.NativeFunction{Name: "error", Arity: -1, Fn: logModuleError},
			"with":    &runtime.NativeFunction{Name: "with", Arity: 1, Fn: logModuleWith},
		},
		Done: true,
	}
}

type loggerBinding struct {
	logger *slog.Logger
	closer io.Closer
}

func logCreate(args []runtime.Value) (runtime.Value, error) {
	if len(args) > 1 {
		return nil, fmt.Errorf("create expects 0 or 1 arguments")
	}

	options := (*runtime.ObjectValue)(nil)
	if len(args) == 1 {
		var ok bool
		options, ok = args[0].(*runtime.ObjectValue)
		if !ok {
			return nil, fmt.Errorf("create expects options object")
		}
	}

	logger, err := newSlogLogger(options)
	if err != nil {
		return nil, err
	}
	return logger.object(), nil
}

func logDefault(args []runtime.Value) (runtime.Value, error) {
	return (&loggerBinding{logger: slog.Default()}).object(), nil
}

func logModuleLog(args []runtime.Value) (runtime.Value, error) {
	return logWithLogger(slog.Default(), args)
}

func logModuleDebug(args []runtime.Value) (runtime.Value, error) {
	return logWithFixedLevel(slog.Default(), slog.LevelDebug, "debug", args)
}

func logModuleInfo(args []runtime.Value) (runtime.Value, error) {
	return logWithFixedLevel(slog.Default(), slog.LevelInfo, "info", args)
}

func logModuleWarn(args []runtime.Value) (runtime.Value, error) {
	return logWithFixedLevel(slog.Default(), slog.LevelWarn, "warn", args)
}

func logModuleError(args []runtime.Value) (runtime.Value, error) {
	return logWithFixedLevel(slog.Default(), slog.LevelError, "error", args)
}

func logModuleWith(args []runtime.Value) (runtime.Value, error) {
	return withLogger(slog.Default(), args)
}

func (binding *loggerBinding) object() *runtime.ObjectValue {
	return &runtime.ObjectValue{Fields: map[string]runtime.Value{
		"log":   &runtime.NativeFunction{Name: "logger.log", Arity: -1, Fn: binding.log},
		"debug": &runtime.NativeFunction{Name: "logger.debug", Arity: -1, Fn: binding.debug},
		"info":  &runtime.NativeFunction{Name: "logger.info", Arity: -1, Fn: binding.info},
		"warn":  &runtime.NativeFunction{Name: "logger.warn", Arity: -1, Fn: binding.warn},
		"error": &runtime.NativeFunction{Name: "logger.error", Arity: -1, Fn: binding.error},
		"with":  &runtime.NativeFunction{Name: "logger.with", Arity: 1, Fn: binding.with},
		"close": &runtime.NativeFunction{Name: "logger.close", Arity: 0, Fn: binding.close},
	}}
}

func (binding *loggerBinding) log(args []runtime.Value) (runtime.Value, error) {
	return logWithLogger(binding.logger, args)
}

func (binding *loggerBinding) debug(args []runtime.Value) (runtime.Value, error) {
	return logWithFixedLevel(binding.logger, slog.LevelDebug, "debug", args)
}

func (binding *loggerBinding) info(args []runtime.Value) (runtime.Value, error) {
	return logWithFixedLevel(binding.logger, slog.LevelInfo, "info", args)
}

func (binding *loggerBinding) warn(args []runtime.Value) (runtime.Value, error) {
	return logWithFixedLevel(binding.logger, slog.LevelWarn, "warn", args)
}

func (binding *loggerBinding) error(args []runtime.Value) (runtime.Value, error) {
	return logWithFixedLevel(binding.logger, slog.LevelError, "error", args)
}

func (binding *loggerBinding) with(args []runtime.Value) (runtime.Value, error) {
	return withLogger(binding.logger, args)
}

func (binding *loggerBinding) close(args []runtime.Value) (runtime.Value, error) {
	if binding.closer == nil {
		return runtime.NullValue{}, nil
	}
	err := binding.closer.Close()
	binding.closer = nil
	return runtime.NullValue{}, err
}

func withLogger(base *slog.Logger, args []runtime.Value) (runtime.Value, error) {
	fields, err := requireLogFieldsArg("with", args)
	if err != nil {
		return nil, err
	}
	attrs, err := runtimeObjectToAttrs(fields)
	if err != nil {
		return nil, err
	}
	items := make([]any, 0, len(attrs))
	for _, attr := range attrs {
		items = append(items, attr)
	}
	return (&loggerBinding{logger: base.With(items...)}).object(), nil
}

func logWithLogger(logger *slog.Logger, args []runtime.Value) (runtime.Value, error) {
	if len(args) < 2 || len(args) > 3 {
		return nil, fmt.Errorf("log expects level, message, and optional fields")
	}
	levelName, err := utils.RequireStringArg("log", args[0])
	if err != nil {
		return nil, err
	}
	level, err := parseLogLevel(levelName)
	if err != nil {
		return nil, err
	}
	message, err := utils.RequireStringArg("log", args[1])
	if err != nil {
		return nil, err
	}
	attrs, err := optionalLogAttrs("log", args[2:])
	if err != nil {
		return nil, err
	}
	logger.Log(nil, level, message, attrsToAny(attrs)...)
	return runtime.NullValue{}, nil
}

func logWithFixedLevel(logger *slog.Logger, level slog.Level, name string, args []runtime.Value) (runtime.Value, error) {
	if len(args) < 1 || len(args) > 2 {
		return nil, fmt.Errorf("%s expects message and optional fields", name)
	}
	message, err := utils.RequireStringArg(name, args[0])
	if err != nil {
		return nil, err
	}
	attrs, err := optionalLogAttrs(name, args[1:])
	if err != nil {
		return nil, err
	}
	logger.Log(nil, level, message, attrsToAny(attrs)...)
	return runtime.NullValue{}, nil
}

func optionalLogAttrs(name string, args []runtime.Value) ([]slog.Attr, error) {
	if len(args) == 0 {
		return nil, nil
	}
	if len(args) > 1 {
		return nil, fmt.Errorf("%s expects message and optional fields", name)
	}
	fields, ok := args[0].(*runtime.ObjectValue)
	if !ok {
		return nil, fmt.Errorf("%s fields must be object", name)
	}
	return runtimeObjectToAttrs(fields)
}

func requireLogFieldsArg(name string, args []runtime.Value) (*runtime.ObjectValue, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("%s expects object argument", name)
	}
	fields, ok := args[0].(*runtime.ObjectValue)
	if !ok {
		return nil, fmt.Errorf("%s expects object argument", name)
	}
	return fields, nil
}

func runtimeObjectToAttrs(value *runtime.ObjectValue) ([]slog.Attr, error) {
	if value == nil {
		return nil, nil
	}
	attrs := make([]slog.Attr, 0, len(value.Fields))
	for key, field := range value.Fields {
		plain, err := utils.RuntimeToPlainValue(field)
		if err != nil {
			return nil, err
		}
		attrs = append(attrs, slog.Any(key, plain))
	}
	return attrs, nil
}

func attrsToAny(attrs []slog.Attr) []any {
	if len(attrs) == 0 {
		return nil
	}
	items := make([]any, 0, len(attrs))
	for _, attr := range attrs {
		items = append(items, attr)
	}
	return items
}

func logLevelsObject() *runtime.ObjectValue {
	return &runtime.ObjectValue{Fields: map[string]runtime.Value{
		"debug": runtime.StringValue{Value: "debug"},
		"info":  runtime.StringValue{Value: "info"},
		"warn":  runtime.StringValue{Value: "warn"},
		"error": runtime.StringValue{Value: "error"},
	}}
}

func newSlogLogger(options *runtime.ObjectValue) (*loggerBinding, error) {
	level := slog.LevelInfo
	format := "text"
	writer := io.Writer(os.Stdout)
	addSource := false
	var closer io.Closer

	if options != nil {
		if value, ok := options.Fields["level"]; ok {
			text, err := utils.RequireStringArg("create", value)
			if err != nil {
				return nil, err
			}
			parsed, err := parseLogLevel(text)
			if err != nil {
				return nil, err
			}
			level = parsed
		}
		if value, ok := options.Fields["format"]; ok {
			text, err := utils.RequireStringArg("create", value)
			if err != nil {
				return nil, err
			}
			switch strings.ToLower(text) {
			case "text", "json":
				format = strings.ToLower(text)
			default:
				return nil, fmt.Errorf("create format must be text or json")
			}
		}
		if value, ok := options.Fields["output"]; ok {
			text, err := utils.RequireStringArg("create", value)
			if err != nil {
				return nil, err
			}
			switch strings.ToLower(text) {
			case "stdout":
				writer = os.Stdout
			case "stderr":
				writer = os.Stderr
			case "file":
				fileWriter, fileCloser, err := newLogFileWriter(options)
				if err != nil {
					return nil, err
				}
				writer = fileWriter
				closer = fileCloser
			default:
				return nil, fmt.Errorf("create output must be stdout, stderr, or file")
			}
		}
		if value, ok := options.Fields["addSource"]; ok {
			boolValue, ok := value.(runtime.BoolValue)
			if !ok {
				return nil, fmt.Errorf("create addSource must be bool")
			}
			addSource = boolValue.Value
		}
	}

	handlerOptions := &slog.HandlerOptions{
		Level:     level,
		AddSource: addSource,
	}
	if format == "json" {
		return &loggerBinding{logger: slog.New(slog.NewJSONHandler(writer, handlerOptions)), closer: closer}, nil
	}
	return &loggerBinding{logger: slog.New(slog.NewTextHandler(writer, handlerOptions)), closer: closer}, nil
}

func newLogFileWriter(options *runtime.ObjectValue) (io.Writer, io.Closer, error) {
	pathValue, ok := options.Fields["filePath"]
	if !ok {
		return nil, nil, fmt.Errorf("create filePath is required when output is file")
	}
	path, err := utils.RequireStringArg("create", pathValue)
	if err != nil {
		return nil, nil, err
	}
	if strings.TrimSpace(path) == "" {
		return nil, nil, fmt.Errorf("create filePath must be non-empty when output is file")
	}

	dir := filepath.Dir(path)
	if dir != "." && dir != "" {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return nil, nil, err
		}
	}

	rotation, ok := options.Fields["rotation"]
	if !ok {
		file, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
		if err != nil {
			return nil, nil, err
		}
		return file, file, nil
	}

	rotationOptions, ok := rotation.(*runtime.ObjectValue)
	if !ok {
		return nil, nil, fmt.Errorf("create rotation must be object")
	}

	maxSize, err := optionalRotationInt(rotationOptions, "maxSizeMB")
	if err != nil {
		return nil, nil, err
	}
	maxBackups, err := optionalRotationInt(rotationOptions, "maxBackups")
	if err != nil {
		return nil, nil, err
	}
	maxAge, err := optionalRotationInt(rotationOptions, "maxAgeDays")
	if err != nil {
		return nil, nil, err
	}
	compress, err := optionalRotationBool(rotationOptions, "compress")
	if err != nil {
		return nil, nil, err
	}
	localTime, err := optionalRotationBool(rotationOptions, "localTime")
	if err != nil {
		return nil, nil, err
	}

	if maxSize <= 0 {
		return nil, nil, fmt.Errorf("create rotation maxSizeMB must be positive")
	}

	logger := &lumberjack.Logger{
		Filename:   path,
		MaxSize:    int(maxSize),
		MaxBackups: int(maxBackups),
		MaxAge:     int(maxAge),
		Compress:   compress,
		LocalTime:  localTime,
	}
	return logger, logger, nil
}

func optionalRotationInt(options *runtime.ObjectValue, name string) (int64, error) {
	if options == nil {
		return 0, nil
	}
	value, ok := options.Fields[name]
	if !ok {
		return 0, nil
	}
	intValue, ok := value.(runtime.IntValue)
	if !ok {
		return 0, fmt.Errorf("create rotation %s must be int", name)
	}
	if intValue.Value < 0 {
		return 0, fmt.Errorf("create rotation %s must be non-negative", name)
	}
	return intValue.Value, nil
}

func optionalRotationBool(options *runtime.ObjectValue, name string) (bool, error) {
	if options == nil {
		return false, nil
	}
	value, ok := options.Fields[name]
	if !ok {
		return false, nil
	}
	boolValue, ok := value.(runtime.BoolValue)
	if !ok {
		return false, fmt.Errorf("create rotation %s must be bool", name)
	}
	return boolValue.Value, nil
}

func parseLogLevel(value string) (slog.Level, error) {
	switch strings.ToLower(value) {
	case "debug":
		return slog.LevelDebug, nil
	case "info":
		return slog.LevelInfo, nil
	case "warn", "warning":
		return slog.LevelWarn, nil
	case "error":
		return slog.LevelError, nil
	default:
		return 0, fmt.Errorf("unsupported log level %q", value)
	}
}
