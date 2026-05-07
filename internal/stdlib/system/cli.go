package system

import (
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"

	langruntime "icoo_lang/internal/runtime"
	"icoo_lang/internal/stdlib/utils"
)

// LoadStdSysCLIModule 加载 std.sys.cli 模块。
func LoadStdSysCLIModule() *langruntime.Module {
	return &langruntime.Module{
		Name: "std.sys.cli",
		Path: "std.sys.cli",
		Exports: map[string]langruntime.Value{
			"create": &langruntime.NativeFunction{Name: "create", Arity: 1, Fn: cliCreate},
		},
		Done: true,
	}
}

type cliFlagSpec struct {
	Name        string
	Short       string
	Description string
	Kind        string
	Default     langruntime.Value
}

type cliCommandBinding struct {
	app         *cliAppBinding
	name        string
	description string
	flags       []*cliFlagSpec
	handler     langruntime.Value
}

type cliAppBinding struct {
	name        string
	description string
	flags       []*cliFlagSpec
	commands    map[string]*cliCommandBinding
	handler     langruntime.Value
}

type cliRunState struct {
	Flags    *langruntime.ObjectValue
	Args     []langruntime.Value
	Raw      []langruntime.Value
	Help     bool
	HelpText string
	Command  string
}

func cliCreate(args []langruntime.Value) (langruntime.Value, error) {
	options, ok := args[0].(*langruntime.ObjectValue)
	if !ok {
		return nil, fmt.Errorf("create expects options object")
	}

	name, err := optionalStringField(options, "name")
	if err != nil {
		return nil, err
	}
	description, err := optionalStringField(options, "description")
	if err != nil {
		return nil, err
	}

	binding := &cliAppBinding{
		name:        name,
		description: description,
		flags:       []*cliFlagSpec{},
		commands:    map[string]*cliCommandBinding{},
	}
	return binding.object(), nil
}

func (binding *cliAppBinding) object() *langruntime.ObjectValue {
	return &langruntime.ObjectValue{Fields: map[string]langruntime.Value{
		"flagString": &langruntime.NativeFunction{Name: "cli.flagString", Arity: 1, Fn: binding.flagString},
		"flagBool":   &langruntime.NativeFunction{Name: "cli.flagBool", Arity: 1, Fn: binding.flagBool},
		"flagInt":    &langruntime.NativeFunction{Name: "cli.flagInt", Arity: 1, Fn: binding.flagInt},
		"command":    &langruntime.NativeFunction{Name: "cli.command", Arity: -1, Fn: binding.command},
		"action":     &langruntime.NativeFunction{Name: "cli.action", Arity: 1, Fn: binding.action},
		"help":       &langruntime.NativeFunction{Name: "cli.help", Arity: 0, Fn: binding.help},
		"run":        &langruntime.NativeFunction{Name: "cli.run", Arity: 0, CtxFn: binding.run},
	}}
}

func (binding *cliCommandBinding) object() *langruntime.ObjectValue {
	return &langruntime.ObjectValue{Fields: map[string]langruntime.Value{
		"flagString": &langruntime.NativeFunction{Name: "cli.command.flagString", Arity: 1, Fn: binding.flagString},
		"flagBool":   &langruntime.NativeFunction{Name: "cli.command.flagBool", Arity: 1, Fn: binding.flagBool},
		"flagInt":    &langruntime.NativeFunction{Name: "cli.command.flagInt", Arity: 1, Fn: binding.flagInt},
		"action":     &langruntime.NativeFunction{Name: "cli.command.action", Arity: 1, Fn: binding.action},
		"help":       &langruntime.NativeFunction{Name: "cli.command.help", Arity: 0, Fn: binding.help},
	}}
}

func (binding *cliAppBinding) flagString(args []langruntime.Value) (langruntime.Value, error) {
	if _, err := binding.addFlag(args[0], "string"); err != nil {
		return nil, err
	}
	return binding.object(), nil
}

func (binding *cliAppBinding) flagBool(args []langruntime.Value) (langruntime.Value, error) {
	if _, err := binding.addFlag(args[0], "bool"); err != nil {
		return nil, err
	}
	return binding.object(), nil
}

func (binding *cliAppBinding) flagInt(args []langruntime.Value) (langruntime.Value, error) {
	if _, err := binding.addFlag(args[0], "int"); err != nil {
		return nil, err
	}
	return binding.object(), nil
}

func (binding *cliCommandBinding) flagString(args []langruntime.Value) (langruntime.Value, error) {
	if _, err := binding.addFlag(args[0], "string"); err != nil {
		return nil, err
	}
	return binding.object(), nil
}

func (binding *cliCommandBinding) flagBool(args []langruntime.Value) (langruntime.Value, error) {
	if _, err := binding.addFlag(args[0], "bool"); err != nil {
		return nil, err
	}
	return binding.object(), nil
}

func (binding *cliCommandBinding) flagInt(args []langruntime.Value) (langruntime.Value, error) {
	if _, err := binding.addFlag(args[0], "int"); err != nil {
		return nil, err
	}
	return binding.object(), nil
}

func (binding *cliAppBinding) addFlag(value langruntime.Value, kind string) (*cliFlagSpec, error) {
	spec, err := parseCLIFlagSpec(value, kind)
	if err != nil {
		return nil, err
	}
	if err := ensureUniqueFlag(binding.flags, spec); err != nil {
		return nil, err
	}
	binding.flags = append(binding.flags, spec)
	return spec, nil
}

func (binding *cliCommandBinding) addFlag(value langruntime.Value, kind string) (*cliFlagSpec, error) {
	spec, err := parseCLIFlagSpec(value, kind)
	if err != nil {
		return nil, err
	}
	if err := ensureUniqueFlag(binding.flags, spec); err != nil {
		return nil, err
	}
	binding.flags = append(binding.flags, spec)
	return spec, nil
}

func (binding *cliAppBinding) command(args []langruntime.Value) (langruntime.Value, error) {
	if len(args) < 1 || len(args) > 2 {
		return nil, fmt.Errorf("command expects options object and optional handler")
	}
	options, ok := args[0].(*langruntime.ObjectValue)
	if !ok {
		return nil, fmt.Errorf("command expects options object")
	}
	name, err := requiredCLIName(options, "name")
	if err != nil {
		return nil, err
	}
	if _, exists := binding.commands[name]; exists {
		return nil, fmt.Errorf("duplicate command: %s", name)
	}
	description, err := optionalStringField(options, "description")
	if err != nil {
		return nil, err
	}
	cmd := &cliCommandBinding{
		app:         binding,
		name:        name,
		description: description,
		flags:       []*cliFlagSpec{},
	}
	if len(args) == 2 {
		cmd.handler = args[1]
	}
	binding.commands[name] = cmd
	return cmd.object(), nil
}

func (binding *cliAppBinding) action(args []langruntime.Value) (langruntime.Value, error) {
	binding.handler = args[0]
	return binding.object(), nil
}

func (binding *cliCommandBinding) action(args []langruntime.Value) (langruntime.Value, error) {
	binding.handler = args[0]
	return binding.object(), nil
}

func (binding *cliAppBinding) help(args []langruntime.Value) (langruntime.Value, error) {
	return langruntime.StringValue{Value: binding.helpText()}, nil
}

func (binding *cliCommandBinding) help(args []langruntime.Value) (langruntime.Value, error) {
	return langruntime.StringValue{Value: binding.helpText()}, nil
}

func (binding *cliAppBinding) run(ctx *langruntime.NativeContext, args []langruntime.Value) (langruntime.Value, error) {
	state, handler, err := binding.parse(os.Args[1:])
	if err != nil {
		return nil, err
	}
	stateValue := state.object()
	if state.Help {
		return stateValue, nil
	}
	if handler != nil {
		if ctx == nil || ctx.CallDetached == nil {
			return nil, fmt.Errorf("cli.run requires runtime call context")
		}
		if _, err := ctx.CallDetached(handler, []langruntime.Value{stateValue}); err != nil {
			return nil, err
		}
	}
	return stateValue, nil
}

func (binding *cliAppBinding) parse(argv []string) (*cliRunState, langruntime.Value, error) {
	rootFlags := defaultFlagObject(binding.flags)
	rootArgs := []langruntime.Value{}
	raw := stringsToRuntimeArray(argv)
	index := 0

	for index < len(argv) {
		token := argv[index]
		if token == "--" {
			for _, item := range argv[index+1:] {
				rootArgs = append(rootArgs, langruntime.StringValue{Value: item})
			}
			index = len(argv)
			break
		}
		if token == "--help" || token == "-h" {
			return &cliRunState{Flags: rootFlags, Args: rootArgs, Raw: raw, Help: true, HelpText: binding.helpText()}, nil, nil
		}
		if command, ok := binding.commands[token]; ok && len(rootArgs) == 0 {
			return binding.parseCommand(command, argv[index+1:], raw, rootFlags, rootArgs)
		}
		if strings.HasPrefix(token, "-") {
			nextIndex, err := parseFlagToken(binding.flags, rootFlags, argv, index)
			if err != nil {
				return nil, nil, err
			}
			index = nextIndex
			continue
		}
		rootArgs = append(rootArgs, langruntime.StringValue{Value: token})
		index++
	}

	state := &cliRunState{
		Flags: rootFlags,
		Args:  rootArgs,
		Raw:   raw,
	}
	return state, binding.handler, nil
}

func (binding *cliAppBinding) parseCommand(command *cliCommandBinding, argv []string, raw []langruntime.Value, rootFlags *langruntime.ObjectValue, rootArgs []langruntime.Value) (*cliRunState, langruntime.Value, error) {
	flags := mergeFlagObjects(rootFlags, defaultFlagObject(command.flags))
	specs := append([]*cliFlagSpec{}, binding.flags...)
	specs = append(specs, command.flags...)
	args := append([]langruntime.Value{}, rootArgs...)
	index := 0

	for index < len(argv) {
		token := argv[index]
		if token == "--" {
			for _, item := range argv[index+1:] {
				args = append(args, langruntime.StringValue{Value: item})
			}
			index = len(argv)
			break
		}
		if token == "--help" || token == "-h" {
			return &cliRunState{Flags: flags, Args: args, Raw: raw, Help: true, HelpText: command.helpText(), Command: command.name}, nil, nil
		}
		if strings.HasPrefix(token, "-") {
			nextIndex, err := parseFlagToken(specs, flags, argv, index)
			if err != nil {
				return nil, nil, err
			}
			index = nextIndex
			continue
		}
		args = append(args, langruntime.StringValue{Value: token})
		index++
	}

	state := &cliRunState{
		Flags:   flags,
		Args:    args,
		Raw:     raw,
		Command: command.name,
	}
	return state, command.handler, nil
}

func (binding *cliAppBinding) helpText() string {
	var sb strings.Builder
	name := binding.name
	if name == "" {
		name = "app"
	}
	sb.WriteString(name)
	if binding.description != "" {
		sb.WriteString(" - ")
		sb.WriteString(binding.description)
	}
	sb.WriteString("\n\nUsage:\n  ")
	sb.WriteString(name)
	if len(binding.flags) > 0 {
		sb.WriteString(" [options]")
	}
	if len(binding.commands) > 0 {
		sb.WriteString(" <command>")
	}
	sb.WriteString(" [args...]\n")
	writeFlagSection(&sb, binding.flags)
	writeCommandSection(&sb, binding.commands)
	return sb.String()
}

func (binding *cliCommandBinding) helpText() string {
	var sb strings.Builder
	appName := binding.app.name
	if appName == "" {
		appName = "app"
	}
	sb.WriteString(appName)
	sb.WriteString(" ")
	sb.WriteString(binding.name)
	if binding.description != "" {
		sb.WriteString(" - ")
		sb.WriteString(binding.description)
	}
	sb.WriteString("\n\nUsage:\n  ")
	sb.WriteString(appName)
	sb.WriteString(" ")
	sb.WriteString(binding.name)
	if len(binding.flags) > 0 {
		sb.WriteString(" [options]")
	}
	sb.WriteString(" [args...]\n")
	writeFlagSection(&sb, binding.flags)
	return sb.String()
}

func (state *cliRunState) object() *langruntime.ObjectValue {
	commandValue := langruntime.Value(langruntime.NullValue{})
	if state.Command != "" {
		commandValue = langruntime.StringValue{Value: state.Command}
	}
	helpTextValue := langruntime.Value(langruntime.NullValue{})
	if state.HelpText != "" {
		helpTextValue = langruntime.StringValue{Value: state.HelpText}
	}
	return &langruntime.ObjectValue{Fields: map[string]langruntime.Value{
		"command":  commandValue,
		"flags":    state.Flags,
		"args":     &langruntime.ArrayValue{Elements: append([]langruntime.Value{}, state.Args...)},
		"raw":      &langruntime.ArrayValue{Elements: append([]langruntime.Value{}, state.Raw...)},
		"help":     langruntime.BoolValue{Value: state.Help},
		"helpText": helpTextValue,
	}}
}

func parseCLIFlagSpec(value langruntime.Value, kind string) (*cliFlagSpec, error) {
	options, ok := value.(*langruntime.ObjectValue)
	if !ok {
		return nil, fmt.Errorf("flag registration expects options object")
	}
	name, err := requiredCLIName(options, "name")
	if err != nil {
		return nil, err
	}
	short, err := optionalStringField(options, "short")
	if err != nil {
		return nil, err
	}
	short = normalizeFlagToken(short)
	if short != "" && strings.Contains(short, "-") {
		return nil, fmt.Errorf("flag short name must be a single token: %s", short)
	}
	description, err := optionalStringField(options, "description")
	if err != nil {
		return nil, err
	}
	defaultValue := defaultValueForKind(kind)
	if rawDefault, ok := options.Fields["default"]; ok {
		if err := validateFlagValue(kind, rawDefault); err != nil {
			return nil, err
		}
		defaultValue = rawDefault
	}
	return &cliFlagSpec{
		Name:        name,
		Short:       short,
		Description: description,
		Kind:        kind,
		Default:     defaultValue,
	}, nil
}

func requiredCLIName(options *langruntime.ObjectValue, field string) (string, error) {
	value, ok := options.Fields[field]
	if !ok {
		return "", fmt.Errorf("%s is required", field)
	}
	name, err := utils.RequireStringArg(field, value)
	if err != nil {
		return "", err
	}
	name = normalizeFlagToken(name)
	if name == "" {
		return "", fmt.Errorf("%s must be non-empty", field)
	}
	return name, nil
}

func optionalStringField(options *langruntime.ObjectValue, field string) (string, error) {
	value, ok := options.Fields[field]
	if !ok {
		return "", nil
	}
	return utils.RequireStringArg(field, value)
}

func ensureUniqueFlag(specs []*cliFlagSpec, next *cliFlagSpec) error {
	for _, current := range specs {
		if current.Name == next.Name {
			return fmt.Errorf("duplicate flag: %s", next.Name)
		}
		if current.Short != "" && current.Short == next.Short {
			return fmt.Errorf("duplicate short flag: %s", next.Short)
		}
	}
	return nil
}

func defaultFlagObject(specs []*cliFlagSpec) *langruntime.ObjectValue {
	fields := map[string]langruntime.Value{}
	for _, spec := range specs {
		fields[spec.Name] = spec.Default
	}
	return &langruntime.ObjectValue{Fields: fields}
}

func mergeFlagObjects(left *langruntime.ObjectValue, right *langruntime.ObjectValue) *langruntime.ObjectValue {
	fields := map[string]langruntime.Value{}
	for key, value := range left.Fields {
		fields[key] = value
	}
	for key, value := range right.Fields {
		fields[key] = value
	}
	return &langruntime.ObjectValue{Fields: fields}
}

func stringsToRuntimeArray(items []string) []langruntime.Value {
	out := make([]langruntime.Value, 0, len(items))
	for _, item := range items {
		out = append(out, langruntime.StringValue{Value: item})
	}
	return out
}

func parseFlagToken(specs []*cliFlagSpec, flags *langruntime.ObjectValue, argv []string, index int) (int, error) {
	token := argv[index]
	nameToken := token
	inlineValue := ""
	if strings.HasPrefix(token, "--") {
		parts := strings.SplitN(token, "=", 2)
		nameToken = parts[0]
		if len(parts) == 2 {
			inlineValue = parts[1]
		}
	}
	spec := findFlagSpec(specs, nameToken)
	if spec == nil {
		return 0, fmt.Errorf("unknown flag: %s", token)
	}
	value, nextIndex, err := parseFlagValue(spec, argv, index, inlineValue)
	if err != nil {
		return 0, err
	}
	flags.Fields[spec.Name] = value
	return nextIndex, nil
}

func findFlagSpec(specs []*cliFlagSpec, token string) *cliFlagSpec {
	normalized := normalizeFlagToken(token)
	for _, spec := range specs {
		if spec.Name == normalized {
			return spec
		}
		if spec.Short != "" && spec.Short == normalized {
			return spec
		}
	}
	return nil
}

func parseFlagValue(spec *cliFlagSpec, argv []string, index int, inlineValue string) (langruntime.Value, int, error) {
	if spec.Kind == "bool" {
		if inlineValue == "" {
			return langruntime.BoolValue{Value: true}, index + 1, nil
		}
		parsed, err := strconv.ParseBool(inlineValue)
		if err != nil {
			return nil, 0, fmt.Errorf("flag %s expects bool value", spec.Name)
		}
		return langruntime.BoolValue{Value: parsed}, index + 1, nil
	}

	valueText := inlineValue
	nextIndex := index + 1
	if valueText == "" {
		if nextIndex >= len(argv) {
			return nil, 0, fmt.Errorf("flag %s expects value", spec.Name)
		}
		valueText = argv[nextIndex]
		nextIndex++
	}

	switch spec.Kind {
	case "string":
		return langruntime.StringValue{Value: valueText}, nextIndex, nil
	case "int":
		parsed, err := strconv.ParseInt(valueText, 10, 64)
		if err != nil {
			return nil, 0, fmt.Errorf("flag %s expects int value", spec.Name)
		}
		return langruntime.IntValue{Value: parsed}, nextIndex, nil
	default:
		return nil, 0, fmt.Errorf("unsupported flag kind: %s", spec.Kind)
	}
}

func validateFlagValue(kind string, value langruntime.Value) error {
	switch kind {
	case "string":
		if _, ok := value.(langruntime.StringValue); ok {
			return nil
		}
	case "bool":
		if _, ok := value.(langruntime.BoolValue); ok {
			return nil
		}
	case "int":
		if _, ok := value.(langruntime.IntValue); ok {
			return nil
		}
	}
	return fmt.Errorf("default value does not match flag kind %s", kind)
}

func defaultValueForKind(kind string) langruntime.Value {
	switch kind {
	case "bool":
		return langruntime.BoolValue{Value: false}
	default:
		return langruntime.NullValue{}
	}
}

func normalizeFlagToken(value string) string {
	value = strings.TrimSpace(value)
	value = strings.TrimLeft(value, "-")
	return value
}

func writeFlagSection(sb *strings.Builder, specs []*cliFlagSpec) {
	if len(specs) == 0 {
		return
	}
	sb.WriteString("\nOptions:\n")
	for _, spec := range specs {
		sb.WriteString("  --")
		sb.WriteString(spec.Name)
		if spec.Short != "" {
			sb.WriteString(", -")
			sb.WriteString(spec.Short)
		}
		if spec.Kind == "string" {
			sb.WriteString(" <string>")
		}
		if spec.Kind == "int" {
			sb.WriteString(" <int>")
		}
		if spec.Description != "" {
			sb.WriteString("  ")
			sb.WriteString(spec.Description)
		}
		if _, isNull := spec.Default.(langruntime.NullValue); !isNull {
			sb.WriteString(" (default: ")
			sb.WriteString(spec.Default.String())
			sb.WriteString(")")
		}
		sb.WriteString("\n")
	}
}

func writeCommandSection(sb *strings.Builder, commands map[string]*cliCommandBinding) {
	if len(commands) == 0 {
		return
	}
	names := make([]string, 0, len(commands))
	for name := range commands {
		names = append(names, name)
	}
	sort.Strings(names)
	sb.WriteString("\nCommands:\n")
	for _, name := range names {
		command := commands[name]
		sb.WriteString("  ")
		sb.WriteString(command.name)
		if command.description != "" {
			sb.WriteString("  ")
			sb.WriteString(command.description)
		}
		sb.WriteString("\n")
	}
}
