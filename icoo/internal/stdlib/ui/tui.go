package ui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	langruntime "icoo_lang/internal/runtime"
)

// LoadStdUITUIModule 加载 std.ui.tui 模块。
func LoadStdUITUIModule() *langruntime.Module {
	return &langruntime.Module{
		Name: "std.ui.tui",
		Path: "std.ui.tui",
		Exports: map[string]langruntime.Value{
			"app":           &langruntime.NativeFunction{Name: "tui.app", Arity: 1, Fn: tuiApp},
			"run":           &langruntime.NativeFunction{Name: "tui.run", Arity: 1, CtxFn: tuiRun},
			"render":        &langruntime.NativeFunction{Name: "tui.render", Arity: 1, Fn: tuiRender},
			"text":          &langruntime.NativeFunction{Name: "tui.text", Arity: 1, Fn: tuiText},
			"box":           &langruntime.NativeFunction{Name: "tui.box", Arity: 1, Fn: tuiBox},
			"vstack":        &langruntime.NativeFunction{Name: "tui.vstack", Arity: 1, Fn: tuiVStack},
			"hstack":        &langruntime.NativeFunction{Name: "tui.hstack", Arity: 1, Fn: tuiHStack},
			"listState":     &langruntime.NativeFunction{Name: "tui.listState", Arity: 1, Fn: tuiListState},
			"listUpdate":    &langruntime.NativeFunction{Name: "tui.listUpdate", Arity: 2, Fn: tuiListUpdate},
			"list":          &langruntime.NativeFunction{Name: "tui.list", Arity: 1, Fn: tuiList},
			"inputState":    &langruntime.NativeFunction{Name: "tui.inputState", Arity: 1, Fn: tuiInputState},
			"inputUpdate":   &langruntime.NativeFunction{Name: "tui.inputUpdate", Arity: 2, Fn: tuiInputUpdate},
			"input":         &langruntime.NativeFunction{Name: "tui.input", Arity: 1, Fn: tuiInput},
			"viewportState": &langruntime.NativeFunction{Name: "tui.viewportState", Arity: 1, Fn: tuiViewportState},
			"viewportUpdate": &langruntime.NativeFunction{
				Name:  "tui.viewportUpdate",
				Arity: 2,
				Fn:    tuiViewportUpdate,
			},
			"viewport":   &langruntime.NativeFunction{Name: "tui.viewport", Arity: 1, Fn: tuiViewport},
			"tableState": &langruntime.NativeFunction{Name: "tui.tableState", Arity: 1, Fn: tuiTableState},
			"tableUpdate": &langruntime.NativeFunction{
				Name:  "tui.tableUpdate",
				Arity: 2,
				Fn:    tuiTableUpdate,
			},
			"table":    &langruntime.NativeFunction{Name: "tui.table", Arity: 1, Fn: tuiTable},
			"progress": &langruntime.NativeFunction{Name: "tui.progress", Arity: 1, Fn: tuiProgress},
			"status":   &langruntime.NativeFunction{Name: "tui.status", Arity: 1, Fn: tuiStatus},
			"spinner":  &langruntime.NativeFunction{Name: "tui.spinner", Arity: 1, Fn: tuiSpinner},
			"quit":     &langruntime.NativeFunction{Name: "tui.quit", Arity: 0, Fn: tuiQuitCommand},
			"emit":     &langruntime.NativeFunction{Name: "tui.emit", Arity: 1, Fn: tuiEmitCommand},
			"tick":     &langruntime.NativeFunction{Name: "tui.tick", Arity: 2, Fn: tuiTickCommandValue},
			"task":     &langruntime.NativeFunction{Name: "tui.task", Arity: 1, Fn: tuiTaskCommand},
		},
		Done: true,
	}
}

type tuiProgram struct {
	ctx         *langruntime.NativeContext
	app         *langruntime.ObjectValue
	model       langruntime.Value
	quitting    bool
	lastMessage *langruntime.ObjectValue
	tickCount   int
	viewCache   string
}

type tuiTickMsg struct {
	Count int
}

type tuiRuntimeMsg struct {
	Value *langruntime.ObjectValue
}

func tuiApp(args []langruntime.Value) (langruntime.Value, error) {
	options, err := requireObject("tui.app", args[0])
	if err != nil {
		return nil, err
	}
	fields := cloneFields(options.Fields)
	fields["__tui_app"] = langruntime.BoolValue{Value: true}
	return &langruntime.ObjectValue{Fields: fields}, nil
}

func tuiRun(ctx *langruntime.NativeContext, args []langruntime.Value) (langruntime.Value, error) {
	if ctx == nil || ctx.CallDetached == nil {
		return nil, fmt.Errorf("tui.run requires runtime call context")
	}
	app, err := requireTUIApp("tui.run", args[0])
	if err != nil {
		return nil, err
	}

	programModel := &tuiProgram{
		ctx: ctx,
		app: app,
	}

	if initValue, ok := app.Fields["init"]; ok {
		model, err := ctx.CallDetached(initValue, nil)
		if err != nil {
			return nil, err
		}
		programModel.model = model
	} else if seed, ok := app.Fields["model"]; ok {
		programModel.model = seed
	} else {
		programModel.model = langruntime.NullValue{}
	}

	if err := programModel.refreshView(); err != nil {
		return nil, err
	}

	options := []tea.ProgramOption{}
	if boolField(app, "altScreen", false) {
		options = append(options, tea.WithAltScreen())
	}
	if boolField(app, "mouse", false) {
		options = append(options, tea.WithMouseCellMotion())
	}

	program := tea.NewProgram(programModel, options...)
	finalModel, err := program.Run()
	if err != nil {
		return nil, err
	}
	finalProgram, ok := finalModel.(*tuiProgram)
	if !ok {
		return nil, fmt.Errorf("unexpected final TUI model")
	}

	return &langruntime.ObjectValue{Fields: map[string]langruntime.Value{
		"model": finalProgram.model,
		"quit":  langruntime.BoolValue{Value: finalProgram.quitting},
	}}, nil
}

func tuiRender(args []langruntime.Value) (langruntime.Value, error) {
	rendered, err := renderTUIView(args[0], 0)
	if err != nil {
		return nil, err
	}
	return langruntime.StringValue{Value: rendered}, nil
}

func tuiText(args []langruntime.Value) (langruntime.Value, error) {
	return buildDescriptor("text", args[0]), nil
}

func tuiBox(args []langruntime.Value) (langruntime.Value, error) {
	return buildDescriptor("box", args[0]), nil
}

func tuiVStack(args []langruntime.Value) (langruntime.Value, error) {
	return buildDescriptor("vstack", args[0]), nil
}

func tuiHStack(args []langruntime.Value) (langruntime.Value, error) {
	return buildDescriptor("hstack", args[0]), nil
}

func tuiList(args []langruntime.Value) (langruntime.Value, error) {
	return buildDescriptor("list", args[0]), nil
}

func tuiInput(args []langruntime.Value) (langruntime.Value, error) {
	return buildDescriptor("input", args[0]), nil
}

func tuiStatus(args []langruntime.Value) (langruntime.Value, error) {
	return buildDescriptor("status", args[0]), nil
}

func tuiSpinner(args []langruntime.Value) (langruntime.Value, error) {
	return buildDescriptor("spinner", args[0]), nil
}

func tuiViewport(args []langruntime.Value) (langruntime.Value, error) {
	return buildDescriptor("viewport", args[0]), nil
}

func tuiTable(args []langruntime.Value) (langruntime.Value, error) {
	return buildDescriptor("table", args[0]), nil
}

func tuiProgress(args []langruntime.Value) (langruntime.Value, error) {
	return buildDescriptor("progress", args[0]), nil
}

func tuiListState(args []langruntime.Value) (langruntime.Value, error) {
	options, err := requireObject("tui.listState", args[0])
	if err != nil {
		return nil, err
	}
	fields := cloneFields(options.Fields)
	fields["items"] = arrayField(options, "items", &langruntime.ArrayValue{Elements: []langruntime.Value{}})
	selected := intField(options, "selected", 0)
	count := len(arrayField(options, "items", &langruntime.ArrayValue{Elements: []langruntime.Value{}}).Elements)
	fields["selected"] = langruntime.IntValue{Value: int64(clampIndex(selected, count))}
	fields["type"] = langruntime.StringValue{Value: "list_state"}
	return &langruntime.ObjectValue{Fields: fields}, nil
}

func tuiListUpdate(args []langruntime.Value) (langruntime.Value, error) {
	state, err := requireObject("tui.listUpdate", args[0])
	if err != nil {
		return nil, err
	}
	msg, err := requireObject("tui.listUpdate", args[1])
	if err != nil {
		return nil, err
	}
	fields := cloneFields(state.Fields)
	items := arrayField(state, "items", &langruntime.ArrayValue{Elements: []langruntime.Value{}})
	selected := clampIndex(intField(state, "selected", 0), len(items.Elements))

	if stringField(msg, "kind", "") == "key" {
		switch stringField(msg, "key", "") {
		case "up", "k":
			selected--
		case "down", "j":
			selected++
		case "home":
			selected = 0
		case "end":
			selected = len(items.Elements) - 1
		case "pgup":
			selected -= 5
		case "pgdown":
			selected += 5
		}
	}
	fields["selected"] = langruntime.IntValue{Value: int64(clampIndex(selected, len(items.Elements)))}
	return &langruntime.ObjectValue{Fields: fields}, nil
}

func tuiInputState(args []langruntime.Value) (langruntime.Value, error) {
	options, err := requireObject("tui.inputState", args[0])
	if err != nil {
		return nil, err
	}
	fields := cloneFields(options.Fields)
	value := stringField(options, "value", "")
	cursor := intField(options, "cursor", len([]rune(value)))
	fields["value"] = langruntime.StringValue{Value: value}
	fields["cursor"] = langruntime.IntValue{Value: int64(clampCursor(cursor, value))}
	fields["placeholder"] = langruntime.StringValue{Value: stringField(options, "placeholder", "")}
	fields["focused"] = langruntime.BoolValue{Value: boolField(options, "focused", true)}
	fields["submitted"] = langruntime.BoolValue{Value: false}
	fields["type"] = langruntime.StringValue{Value: "input_state"}
	return &langruntime.ObjectValue{Fields: fields}, nil
}

func tuiInputUpdate(args []langruntime.Value) (langruntime.Value, error) {
	state, err := requireObject("tui.inputUpdate", args[0])
	if err != nil {
		return nil, err
	}
	msg, err := requireObject("tui.inputUpdate", args[1])
	if err != nil {
		return nil, err
	}

	fields := cloneFields(state.Fields)
	value := stringField(state, "value", "")
	cursor := clampCursor(intField(state, "cursor", len([]rune(value))), value)
	focused := boolField(state, "focused", true)
	submitted := false

	if focused && stringField(msg, "kind", "") == "key" {
		key := stringField(msg, "key", "")
		text := stringField(msg, "text", "")
		switch key {
		case "left":
			cursor--
		case "right":
			cursor++
		case "home", "ctrl+a":
			cursor = 0
		case "end", "ctrl+e":
			cursor = len([]rune(value))
		case "backspace", "ctrl+h":
			value, cursor = deleteBeforeCursor(value, cursor)
		case "delete":
			value, cursor = deleteAtCursor(value, cursor)
		case "ctrl+u":
			value, cursor = "", 0
		case "enter":
			submitted = true
		default:
			if text != "" {
				value, cursor = insertAtCursor(value, cursor, text)
			}
		}
	}

	cursor = clampCursor(cursor, value)
	fields["value"] = langruntime.StringValue{Value: value}
	fields["cursor"] = langruntime.IntValue{Value: int64(cursor)}
	fields["submitted"] = langruntime.BoolValue{Value: submitted}
	return &langruntime.ObjectValue{Fields: fields}, nil
}

func tuiViewportState(args []langruntime.Value) (langruntime.Value, error) {
	options, err := requireObject("tui.viewportState", args[0])
	if err != nil {
		return nil, err
	}
	fields := cloneFields(options.Fields)
	fields["content"] = langruntime.StringValue{Value: stringField(options, "content", "")}
	fields["height"] = langruntime.IntValue{Value: int64(maxInt(intField(options, "height", 8), 1))}
	fields["offset"] = langruntime.IntValue{Value: int64(maxInt(intField(options, "offset", 0), 0))}
	fields["type"] = langruntime.StringValue{Value: "viewport_state"}
	return &langruntime.ObjectValue{Fields: fields}, nil
}

func tuiViewportUpdate(args []langruntime.Value) (langruntime.Value, error) {
	state, err := requireObject("tui.viewportUpdate", args[0])
	if err != nil {
		return nil, err
	}
	msg, err := requireObject("tui.viewportUpdate", args[1])
	if err != nil {
		return nil, err
	}
	fields := cloneFields(state.Fields)
	content := stringField(state, "content", "")
	height := maxInt(intField(state, "height", 8), 1)
	offset := maxInt(intField(state, "offset", 0), 0)
	maxOffset := maxViewportOffset(content, height)

	if stringField(msg, "kind", "") == "key" {
		switch stringField(msg, "key", "") {
		case "up", "k":
			offset--
		case "down", "j":
			offset++
		case "pgup":
			offset -= height
		case "pgdown":
			offset += height
		case "home":
			offset = 0
		case "end":
			offset = maxOffset
		}
	}
	fields["offset"] = langruntime.IntValue{Value: int64(clampRange(offset, 0, maxOffset))}
	return &langruntime.ObjectValue{Fields: fields}, nil
}

func tuiTableState(args []langruntime.Value) (langruntime.Value, error) {
	options, err := requireObject("tui.tableState", args[0])
	if err != nil {
		return nil, err
	}
	fields := cloneFields(options.Fields)
	rows := arrayField(options, "rows", &langruntime.ArrayValue{Elements: []langruntime.Value{}})
	height := maxInt(intField(options, "height", 8), 1)
	selected := clampIndex(intField(options, "selected", 0), len(rows.Elements))
	fields["columns"] = arrayField(options, "columns", &langruntime.ArrayValue{Elements: []langruntime.Value{}})
	fields["rows"] = rows
	fields["height"] = langruntime.IntValue{Value: int64(height)}
	fields["selected"] = langruntime.IntValue{Value: int64(selected)}
	fields["offset"] = langruntime.IntValue{Value: int64(clampRange(intField(options, "offset", 0), 0, maxInt(len(rows.Elements)-height, 0)))}
	fields["type"] = langruntime.StringValue{Value: "table_state"}
	return &langruntime.ObjectValue{Fields: fields}, nil
}

func tuiTableUpdate(args []langruntime.Value) (langruntime.Value, error) {
	state, err := requireObject("tui.tableUpdate", args[0])
	if err != nil {
		return nil, err
	}
	msg, err := requireObject("tui.tableUpdate", args[1])
	if err != nil {
		return nil, err
	}
	fields := cloneFields(state.Fields)
	rows := arrayField(state, "rows", &langruntime.ArrayValue{Elements: []langruntime.Value{}})
	height := maxInt(intField(state, "height", 8), 1)
	selected := clampIndex(intField(state, "selected", 0), len(rows.Elements))

	if stringField(msg, "kind", "") == "key" {
		switch stringField(msg, "key", "") {
		case "up", "k":
			selected--
		case "down", "j":
			selected++
		case "home":
			selected = 0
		case "end":
			selected = len(rows.Elements) - 1
		case "pgup":
			selected -= height
		case "pgdown":
			selected += height
		}
	}

	selected = clampIndex(selected, len(rows.Elements))
	offset := clampViewportOffset(intField(state, "offset", 0), selected, height, len(rows.Elements))
	fields["selected"] = langruntime.IntValue{Value: int64(selected)}
	fields["offset"] = langruntime.IntValue{Value: int64(offset)}
	return &langruntime.ObjectValue{Fields: fields}, nil
}

func tuiQuitCommand(args []langruntime.Value) (langruntime.Value, error) {
	return &langruntime.ObjectValue{Fields: map[string]langruntime.Value{
		"type": langruntime.StringValue{Value: "quit_command"},
	}}, nil
}

func tuiEmitCommand(args []langruntime.Value) (langruntime.Value, error) {
	message, err := messageObjectFromValue(args[0], "emit")
	if err != nil {
		return nil, err
	}
	return &langruntime.ObjectValue{Fields: map[string]langruntime.Value{
		"type":    langruntime.StringValue{Value: "emit_command"},
		"message": message,
	}}, nil
}

func tuiTickCommandValue(args []langruntime.Value) (langruntime.Value, error) {
	delay, ok := args[0].(langruntime.IntValue)
	if !ok {
		return nil, fmt.Errorf("tui.tick expects delay milliseconds as int")
	}
	message, err := messageObjectFromValue(args[1], "tick")
	if err != nil {
		return nil, err
	}
	return &langruntime.ObjectValue{Fields: map[string]langruntime.Value{
		"type":      langruntime.StringValue{Value: "tick_command"},
		"delayMs":   delay,
		"message":   message,
		"timestamp": langruntime.IntValue{Value: int64(time.Now().UnixMilli())},
	}}, nil
}

func tuiTaskCommand(args []langruntime.Value) (langruntime.Value, error) {
	return &langruntime.ObjectValue{Fields: map[string]langruntime.Value{
		"type": langruntime.StringValue{Value: "task_command"},
		"run":  args[0],
	}}, nil
}

func (p *tuiProgram) Init() tea.Cmd {
	return tuiTickCmd(1)
}

func (p *tuiProgram) Update(message tea.Msg) (tea.Model, tea.Cmd) {
	cmds := []tea.Cmd{}
	msgValue := messageToRuntimeValue(message, p.tickCount)
	if msgValue != nil {
		p.lastMessage = msgValue
		resultCommands, err := p.callUpdate(msgValue)
		if err != nil {
			p.viewCache = "TUI runtime error: " + err.Error()
			p.quitting = true
			return p, tea.Quit
		}
		cmds = append(cmds, resultCommands...)
	}

	switch msg := message.(type) {
	case tea.KeyMsg:
		if msg.String() == "ctrl+c" && !p.quitting {
			p.quitting = true
			return p, tea.Quit
		}
	case tuiTickMsg:
		p.tickCount = msg.Count
		cmds = append(cmds, tuiTickCmd(msg.Count+1))
	case tuiRuntimeMsg:
		_ = msg
	case tea.QuitMsg:
		p.quitting = true
	}

	if p.quitting {
		return p, tea.Quit
	}
	if err := p.refreshView(); err != nil {
		p.viewCache = "TUI render error: " + err.Error()
		p.quitting = true
		return p, tea.Quit
	}
	if len(cmds) == 0 {
		return p, nil
	}
	return p, tea.Batch(cmds...)
}

func (p *tuiProgram) View() string {
	return p.viewCache
}

func (p *tuiProgram) callUpdate(msg *langruntime.ObjectValue) ([]tea.Cmd, error) {
	updateValue, ok := p.app.Fields["update"]
	if !ok {
		return nil, nil
	}
	result, err := p.ctx.CallDetached(updateValue, []langruntime.Value{p.model, msg})
	if err != nil {
		return nil, err
	}
	switch value := result.(type) {
	case *langruntime.ObjectValue:
		if model, ok := value.Fields["model"]; ok {
			p.model = model
		} else {
			p.model = value
		}
		if quit := boolObjectField(value, "quit", false); quit {
			p.quitting = true
		}
		return p.commandsFromResult(value)
	default:
		p.model = result
	}
	return nil, nil
}

func (p *tuiProgram) refreshView() error {
	viewValue, ok := p.app.Fields["view"]
	if !ok {
		p.viewCache = ""
		return nil
	}
	renderable, err := p.ctx.CallDetached(viewValue, []langruntime.Value{p.model})
	if err != nil {
		return err
	}
	rendered, err := renderTUIView(renderable, p.tickCount)
	if err != nil {
		return err
	}
	p.viewCache = rendered
	return nil
}

func tuiTickCmd(next int) tea.Cmd {
	return tea.Tick(120*time.Millisecond, func(time.Time) tea.Msg {
		return tuiTickMsg{Count: next}
	})
}

func messageToRuntimeValue(message tea.Msg, tick int) *langruntime.ObjectValue {
	switch msg := message.(type) {
	case tea.KeyMsg:
		text := ""
		if len(msg.Runes) > 0 {
			text = string(msg.Runes)
		}
		return &langruntime.ObjectValue{Fields: map[string]langruntime.Value{
			"kind":  langruntime.StringValue{Value: "key"},
			"key":   langruntime.StringValue{Value: msg.String()},
			"text":  langruntime.StringValue{Value: text},
			"alt":   langruntime.BoolValue{Value: msg.Alt},
			"paste": langruntime.BoolValue{Value: msg.Paste},
		}}
	case tea.WindowSizeMsg:
		return &langruntime.ObjectValue{Fields: map[string]langruntime.Value{
			"kind":   langruntime.StringValue{Value: "window"},
			"width":  langruntime.IntValue{Value: int64(msg.Width)},
			"height": langruntime.IntValue{Value: int64(msg.Height)},
		}}
	case tuiTickMsg:
		return &langruntime.ObjectValue{Fields: map[string]langruntime.Value{
			"kind":  langruntime.StringValue{Value: "tick"},
			"count": langruntime.IntValue{Value: int64(msg.Count)},
			"frame": langruntime.IntValue{Value: int64(tick)},
		}}
	case tuiRuntimeMsg:
		return msg.Value
	default:
		return nil
	}
}

func renderTUIView(value langruntime.Value, tick int) (string, error) {
	switch current := value.(type) {
	case nil:
		return "", nil
	case langruntime.NullValue:
		return "", nil
	case langruntime.StringValue:
		return current.Value, nil
	case *langruntime.ArrayValue:
		lines := make([]string, 0, len(current.Elements))
		for _, child := range current.Elements {
			rendered, err := renderTUIView(child, tick)
			if err != nil {
				return "", err
			}
			lines = append(lines, rendered)
		}
		return strings.Join(lines, "\n"), nil
	case *langruntime.ObjectValue:
		switch stringField(current, "type", "") {
		case "text":
			return renderTextNode(current)
		case "box":
			return renderBoxNode(current, tick)
		case "vstack":
			return renderStackNode(current, tick, true)
		case "hstack":
			return renderStackNode(current, tick, false)
		case "list":
			return renderListNode(current, tick)
		case "input":
			return renderInputNode(current, tick)
		case "viewport":
			return renderViewportNode(current, tick)
		case "table":
			return renderTableNode(current, tick)
		case "progress":
			return renderProgressNode(current)
		case "status":
			return renderStatusNode(current)
		case "spinner":
			return renderSpinnerNode(current, tick)
		default:
			return current.String(), nil
		}
	default:
		return current.String(), nil
	}
}

func renderTextNode(node *langruntime.ObjectValue) (string, error) {
	content := stringField(node, "content", stringField(node, "text", ""))
	return styleFromNode(node).Render(content), nil
}

func renderBoxNode(node *langruntime.ObjectValue, tick int) (string, error) {
	child, _ := node.Fields["child"]
	content, err := renderTUIView(child, tick)
	if err != nil {
		return "", err
	}
	title := stringField(node, "title", "")
	if title != "" {
		content = title + "\n" + content
	}
	return styleFromNode(node).Render(content), nil
}

func renderStackNode(node *langruntime.ObjectValue, tick int, vertical bool) (string, error) {
	children := arrayField(node, "children", &langruntime.ArrayValue{Elements: []langruntime.Value{}})
	parts := make([]string, 0, len(children.Elements))
	for _, child := range children.Elements {
		rendered, err := renderTUIView(child, tick)
		if err != nil {
			return "", err
		}
		parts = append(parts, rendered)
	}
	gap := intField(node, "gap", 0)
	joined := ""
	if vertical {
		separator := "\n"
		if gap > 0 {
			separator = strings.Repeat("\n", gap+1)
		}
		joined = strings.Join(parts, separator)
	} else {
		separator := ""
		if gap > 0 {
			separator = strings.Repeat(" ", gap)
		}
		joined = lipgloss.JoinHorizontal(lipgloss.Top, interleave(parts, separator)...)
	}
	return styleFromNode(node).Render(joined), nil
}

func renderListNode(node *langruntime.ObjectValue, tick int) (string, error) {
	state := objectField(node, "state", node)
	items := arrayField(state, "items", arrayField(node, "items", &langruntime.ArrayValue{Elements: []langruntime.Value{}}))
	selected := clampIndex(intField(state, "selected", intField(node, "selected", 0)), len(items.Elements))
	marker := stringField(node, "selectedMarker", "›")
	itemStyle := styleFromField(node, "itemStyle")
	selectedStyle, hasSelectedStyle := styleFromFieldMaybe(node, "selectedStyle")
	if !hasSelectedStyle {
		selectedStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("205")).Bold(true)
	}

	lines := make([]string, 0, len(items.Elements))
	for index, item := range items.Elements {
		label, err := renderTUIView(item, tick)
		if err != nil {
			return "", err
		}
		if index == selected {
			lines = append(lines, selectedStyle.Render(marker+" "+label))
			continue
		}
		lines = append(lines, itemStyle.Render("  "+label))
	}
	return styleFromNode(node).Render(strings.Join(lines, "\n")), nil
}

func renderInputNode(node *langruntime.ObjectValue, tick int) (string, error) {
	state := objectField(node, "state", node)
	value := stringField(state, "value", stringField(node, "value", ""))
	placeholder := stringField(state, "placeholder", stringField(node, "placeholder", ""))
	cursor := clampCursor(intField(state, "cursor", len([]rune(value))), value)
	focused := boolField(state, "focused", boolField(node, "focused", true))
	cursorToken := stringField(node, "cursor", "│")
	inputStyle := styleFromNode(node)
	placeholderStyle, hasPlaceholderStyle := styleFromFieldMaybe(node, "placeholderStyle")
	if !hasPlaceholderStyle {
		placeholderStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
	}

	display := value
	if display == "" {
		display = placeholderStyle.Render(placeholder)
	} else if focused {
		display = withCursor(value, cursor, cursorToken)
	}
	return inputStyle.Render(display), nil
}

func renderViewportNode(node *langruntime.ObjectValue, tick int) (string, error) {
	state := objectField(node, "state", node)
	content := stringField(state, "content", stringField(node, "content", ""))
	if child, ok := node.Fields["child"]; ok {
		rendered, err := renderTUIView(child, tick)
		if err != nil {
			return "", err
		}
		content = rendered
	}
	height := maxInt(intField(state, "height", intField(node, "height", 8)), 1)
	offset := clampRange(intField(state, "offset", intField(node, "offset", 0)), 0, maxViewportOffset(content, height))
	lines := strings.Split(strings.ReplaceAll(content, "\r\n", "\n"), "\n")
	end := minInt(offset+height, len(lines))
	visible := []string{}
	if offset < len(lines) {
		visible = lines[offset:end]
	}
	return styleFromNode(node).Render(strings.Join(visible, "\n")), nil
}

func renderTableNode(node *langruntime.ObjectValue, tick int) (string, error) {
	state := objectField(node, "state", node)
	columns := arrayField(state, "columns", arrayField(node, "columns", &langruntime.ArrayValue{Elements: []langruntime.Value{}}))
	rows := arrayField(state, "rows", arrayField(node, "rows", &langruntime.ArrayValue{Elements: []langruntime.Value{}}))
	height := maxInt(intField(state, "height", intField(node, "height", 8)), 1)
	selected := clampIndex(intField(state, "selected", intField(node, "selected", 0)), len(rows.Elements))
	offset := clampViewportOffset(intField(state, "offset", intField(node, "offset", 0)), selected, height, len(rows.Elements))

	headerCells := make([]string, 0, len(columns.Elements))
	for _, column := range columns.Elements {
		headerCells = append(headerCells, renderTableCell(column))
	}
	lines := []string{}
	if len(headerCells) > 0 {
		lines = append(lines, lipgloss.NewStyle().Bold(true).Render(strings.Join(headerCells, " | ")))
		lines = append(lines, strings.Repeat("-", maxInt(len(strings.Join(headerCells, " | ")), 3)))
	}

	end := minInt(offset+height, len(rows.Elements))
	for index := offset; index < end; index++ {
		row := rows.Elements[index]
		rowText := renderTableRow(row)
		if index == selected {
			rowText = lipgloss.NewStyle().Foreground(lipgloss.Color("205")).Bold(true).Render("› " + rowText)
		} else {
			rowText = "  " + rowText
		}
		lines = append(lines, rowText)
	}
	return styleFromNode(node).Render(strings.Join(lines, "\n")), nil
}

func renderProgressNode(node *langruntime.ObjectValue) (string, error) {
	value := float64(intField(node, "value", 0))
	total := float64(intField(node, "total", 100))
	if total <= 0 {
		total = 100
	}
	ratio := value / total
	if ratio < 0 {
		ratio = 0
	}
	if ratio > 1 {
		ratio = 1
	}
	width := maxInt(intField(node, "width", 24), 4)
	filled := int(ratio * float64(width))
	bar := strings.Repeat("█", filled) + strings.Repeat("░", width-filled)
	label := stringField(node, "label", "")
	showPercent := boolField(node, "showPercent", true)
	text := "[" + bar + "]"
	if showPercent {
		text += fmt.Sprintf(" %3.0f%%", ratio*100)
	}
	if label != "" {
		text = label + " " + text
	}
	return styleFromNode(node).Render(text), nil
}

func renderStatusNode(node *langruntime.ObjectValue) (string, error) {
	text := stringField(node, "text", stringField(node, "content", ""))
	tone := stringField(node, "tone", "info")
	style := lipgloss.NewStyle().Padding(0, 1).Bold(true)
	switch tone {
	case "success":
		style = style.Foreground(lipgloss.Color("230")).Background(lipgloss.Color("28"))
	case "warning":
		style = style.Foreground(lipgloss.Color("232")).Background(lipgloss.Color("220"))
	case "error":
		style = style.Foreground(lipgloss.Color("230")).Background(lipgloss.Color("160"))
	default:
		style = style.Foreground(lipgloss.Color("230")).Background(lipgloss.Color("62"))
	}
	return style.Copy().Inherit(styleFromNode(node)).Render(text), nil
}

func renderSpinnerNode(node *langruntime.ObjectValue, tick int) (string, error) {
	text := stringField(node, "text", "Loading")
	frameSet := spinnerByName(stringField(node, "frame", "dot"))
	frame := frameSet.Frames[0]
	if len(frameSet.Frames) > 0 {
		frame = frameSet.Frames[tick%len(frameSet.Frames)]
	}
	return styleFromNode(node).Render(frame + " " + text), nil
}

func spinnerByName(name string) spinner.Spinner {
	switch name {
	case "line":
		return spinner.Line
	case "miniDot":
		return spinner.MiniDot
	case "jump":
		return spinner.Jump
	case "pulse":
		return spinner.Pulse
	case "points":
		return spinner.Points
	default:
		return spinner.Dot
	}
}

func buildDescriptor(kind string, value langruntime.Value) *langruntime.ObjectValue {
	if object, ok := value.(*langruntime.ObjectValue); ok {
		fields := cloneFields(object.Fields)
		fields["type"] = langruntime.StringValue{Value: kind}
		return &langruntime.ObjectValue{Fields: fields}
	}
	text := ""
	switch current := value.(type) {
	case langruntime.StringValue:
		text = current.Value
	default:
		if value != nil {
			text = value.String()
		}
	}
	return &langruntime.ObjectValue{Fields: map[string]langruntime.Value{
		"type":    langruntime.StringValue{Value: kind},
		"content": langruntime.StringValue{Value: text},
		"text":    langruntime.StringValue{Value: text},
	}}
}

func requireTUIApp(name string, value langruntime.Value) (*langruntime.ObjectValue, error) {
	object, err := requireObject(name, value)
	if err != nil {
		return nil, err
	}
	if !boolField(object, "__tui_app", false) {
		return nil, fmt.Errorf("%s expects value returned by tui.app", name)
	}
	return object, nil
}

func requireObject(name string, value langruntime.Value) (*langruntime.ObjectValue, error) {
	object, ok := value.(*langruntime.ObjectValue)
	if !ok {
		return nil, fmt.Errorf("%s expects object", name)
	}
	return object, nil
}

func cloneFields(fields map[string]langruntime.Value) map[string]langruntime.Value {
	cloned := make(map[string]langruntime.Value, len(fields))
	for key, value := range fields {
		cloned[key] = value
	}
	return cloned
}

func stringField(object *langruntime.ObjectValue, key string, fallback string) string {
	if object == nil {
		return fallback
	}
	value, ok := object.Fields[key]
	if !ok {
		return fallback
	}
	text, ok := value.(langruntime.StringValue)
	if !ok {
		return fallback
	}
	return text.Value
}

func intField(object *langruntime.ObjectValue, key string, fallback int) int {
	if object == nil {
		return fallback
	}
	value, ok := object.Fields[key]
	if !ok {
		return fallback
	}
	switch current := value.(type) {
	case langruntime.IntValue:
		return int(current.Value)
	case langruntime.FloatValue:
		return int(current.Value)
	default:
		return fallback
	}
}

func boolField(object *langruntime.ObjectValue, key string, fallback bool) bool {
	if object == nil {
		return fallback
	}
	value, ok := object.Fields[key]
	if !ok {
		return fallback
	}
	flag, ok := value.(langruntime.BoolValue)
	if !ok {
		return fallback
	}
	return flag.Value
}

func boolObjectField(object *langruntime.ObjectValue, key string, fallback bool) bool {
	return boolField(object, key, fallback)
}

func arrayField(object *langruntime.ObjectValue, key string, fallback *langruntime.ArrayValue) *langruntime.ArrayValue {
	if object == nil {
		return fallback
	}
	value, ok := object.Fields[key]
	if !ok {
		return fallback
	}
	array, ok := value.(*langruntime.ArrayValue)
	if !ok {
		return fallback
	}
	return array
}

func objectField(object *langruntime.ObjectValue, key string, fallback *langruntime.ObjectValue) *langruntime.ObjectValue {
	if object == nil {
		return fallback
	}
	value, ok := object.Fields[key]
	if !ok {
		return fallback
	}
	nested, ok := value.(*langruntime.ObjectValue)
	if !ok {
		return fallback
	}
	return nested
}

func styleFromField(node *langruntime.ObjectValue, key string) lipgloss.Style {
	return styleFromNode(objectField(node, key, nil))
}

func styleFromFieldMaybe(node *langruntime.ObjectValue, key string) (lipgloss.Style, bool) {
	nested := objectField(node, key, nil)
	if nested == nil {
		return lipgloss.NewStyle(), false
	}
	return styleFromNode(nested), true
}

func (p *tuiProgram) commandsFromResult(result *langruntime.ObjectValue) ([]tea.Cmd, error) {
	cmds := []tea.Cmd{}
	if command := objectField(result, "command", nil); command != nil {
		cmd, err := p.commandFromValue(command)
		if err != nil {
			return nil, err
		}
		if cmd != nil {
			cmds = append(cmds, cmd)
		}
	}
	if commands, ok := result.Fields["commands"]; ok {
		normalized, err := p.commandsFromValue(commands)
		if err != nil {
			return nil, err
		}
		cmds = append(cmds, normalized...)
	}
	return cmds, nil
}

func (p *tuiProgram) commandsFromValue(value langruntime.Value) ([]tea.Cmd, error) {
	switch current := value.(type) {
	case *langruntime.ArrayValue:
		cmds := make([]tea.Cmd, 0, len(current.Elements))
		for _, item := range current.Elements {
			cmd, err := p.commandFromValue(item)
			if err != nil {
				return nil, err
			}
			if cmd != nil {
				cmds = append(cmds, cmd)
			}
		}
		return cmds, nil
	case *langruntime.ObjectValue:
		cmd, err := p.commandFromValue(current)
		if err != nil {
			return nil, err
		}
		if cmd == nil {
			return nil, nil
		}
		return []tea.Cmd{cmd}, nil
	case langruntime.NullValue:
		return nil, nil
	default:
		return nil, fmt.Errorf("tui commands expects object or array")
	}
}

func (p *tuiProgram) commandFromValue(value langruntime.Value) (tea.Cmd, error) {
	command, ok := value.(*langruntime.ObjectValue)
	if !ok {
		return nil, nil
	}
	switch stringField(command, "type", "") {
	case "quit_command":
		return tea.Quit, nil
	case "emit_command":
		message := objectField(command, "message", nil)
		return func() tea.Msg {
			return tuiRuntimeMsg{Value: message}
		}, nil
	case "tick_command":
		delayMs := maxInt(intField(command, "delayMs", 0), 0)
		message := objectField(command, "message", nil)
		return tea.Tick(time.Duration(delayMs)*time.Millisecond, func(time.Time) tea.Msg {
			return tuiRuntimeMsg{Value: message}
		}), nil
	case "task_command":
		if p.ctx == nil || p.ctx.CallDetached == nil {
			return nil, fmt.Errorf("tui task command requires runtime context")
		}
		runValue, ok := command.Fields["run"]
		if !ok {
			return nil, fmt.Errorf("tui task command requires run callback")
		}
		return func() tea.Msg {
			result, err := p.ctx.CallDetached(runValue, nil)
			if err != nil {
				return tuiRuntimeMsg{Value: &langruntime.ObjectValue{Fields: map[string]langruntime.Value{
					"kind":  langruntime.StringValue{Value: "task_error"},
					"error": langruntime.StringValue{Value: err.Error()},
				}}}
			}
			message, convertErr := messageObjectFromValue(result, "task")
			if convertErr != nil {
				return tuiRuntimeMsg{Value: &langruntime.ObjectValue{Fields: map[string]langruntime.Value{
					"kind":  langruntime.StringValue{Value: "task_error"},
					"error": langruntime.StringValue{Value: convertErr.Error()},
				}}}
			}
			return tuiRuntimeMsg{Value: message}
		}, nil
	default:
		return nil, nil
	}
}

func messageObjectFromValue(value langruntime.Value, source string) (*langruntime.ObjectValue, error) {
	object, ok := value.(*langruntime.ObjectValue)
	if ok {
		return object, nil
	}
	return &langruntime.ObjectValue{Fields: map[string]langruntime.Value{
		"kind":   langruntime.StringValue{Value: source},
		"value":  value,
		"source": langruntime.StringValue{Value: source},
	}}, nil
}

func styleFromNode(node *langruntime.ObjectValue) lipgloss.Style {
	style := lipgloss.NewStyle()
	if node == nil {
		return style
	}
	styleObject := objectField(node, "style", nil)
	if styleObject != nil {
		node = styleObject
	}

	if color := stringField(node, "foreground", ""); color != "" {
		style = style.Foreground(lipgloss.Color(color))
	}
	if color := stringField(node, "background", ""); color != "" {
		style = style.Background(lipgloss.Color(color))
	}
	if boolField(node, "bold", false) {
		style = style.Bold(true)
	}
	if boolField(node, "italic", false) {
		style = style.Italic(true)
	}
	if boolField(node, "underline", false) {
		style = style.Underline(true)
	}
	if width := intField(node, "width", 0); width > 0 {
		style = style.Width(width)
	}
	if height := intField(node, "height", 0); height > 0 {
		style = style.Height(height)
	}
	switch stringField(node, "align", "") {
	case "center":
		style = style.Align(lipgloss.Center)
	case "right":
		style = style.Align(lipgloss.Right)
	}
	if border := stringField(node, "border", ""); border != "" {
		style = style.Border(borderByName(border))
	}
	if values := spacingValues(node, "padding"); len(values) > 0 {
		style = style.Padding(values[0], values[1], values[2], values[3])
	}
	if values := spacingValues(node, "margin"); len(values) > 0 {
		style = style.Margin(values[0], values[1], values[2], values[3])
	}
	return style
}

func spacingValues(node *langruntime.ObjectValue, key string) []int {
	if node == nil {
		return nil
	}
	value, ok := node.Fields[key]
	if !ok {
		return nil
	}
	switch current := value.(type) {
	case langruntime.IntValue:
		number := int(current.Value)
		return []int{number, number, number, number}
	case *langruntime.ArrayValue:
		values := make([]int, 0, len(current.Elements))
		for _, element := range current.Elements {
			if number, ok := element.(langruntime.IntValue); ok {
				values = append(values, int(number.Value))
			}
		}
		switch len(values) {
		case 1:
			return []int{values[0], values[0], values[0], values[0]}
		case 2:
			return []int{values[0], values[1], values[0], values[1]}
		case 4:
			return values
		}
	}
	return nil
}

func borderByName(name string) lipgloss.Border {
	switch name {
	case "rounded":
		return lipgloss.RoundedBorder()
	case "double":
		return lipgloss.DoubleBorder()
	case "thick":
		return lipgloss.ThickBorder()
	case "hidden":
		return lipgloss.HiddenBorder()
	default:
		return lipgloss.NormalBorder()
	}
}

func interleave(parts []string, separator string) []string {
	if len(parts) == 0 || separator == "" {
		return parts
	}
	items := make([]string, 0, len(parts)*2-1)
	for index, part := range parts {
		if index > 0 {
			items = append(items, separator)
		}
		items = append(items, part)
	}
	return items
}

func renderTableCell(value langruntime.Value) string {
	switch current := value.(type) {
	case langruntime.StringValue:
		return current.Value
	default:
		if current != nil {
			return current.String()
		}
		return ""
	}
}

func renderTableRow(value langruntime.Value) string {
	if object, ok := value.(*langruntime.ObjectValue); ok {
		if cells := arrayField(object, "cells", nil); cells != nil {
			parts := make([]string, 0, len(cells.Elements))
			for _, cell := range cells.Elements {
				parts = append(parts, renderTableCell(cell))
			}
			return strings.Join(parts, " | ")
		}
	}
	if row, ok := value.(*langruntime.ArrayValue); ok {
		parts := make([]string, 0, len(row.Elements))
		for _, cell := range row.Elements {
			parts = append(parts, renderTableCell(cell))
		}
		return strings.Join(parts, " | ")
	}
	if value == nil {
		return ""
	}
	return value.String()
}

func clampIndex(index int, count int) int {
	if count <= 0 {
		return 0
	}
	if index < 0 {
		return 0
	}
	if index >= count {
		return count - 1
	}
	return index
}

func clampRange(value int, minValue int, maxValue int) int {
	if value < minValue {
		return minValue
	}
	if value > maxValue {
		return maxValue
	}
	return value
}

func maxViewportOffset(content string, height int) int {
	lines := strings.Split(strings.ReplaceAll(content, "\r\n", "\n"), "\n")
	return maxInt(len(lines)-height, 0)
}

func clampViewportOffset(offset int, selected int, height int, count int) int {
	maxOffset := maxInt(count-height, 0)
	offset = clampRange(offset, 0, maxOffset)
	if selected < offset {
		offset = selected
	}
	if selected >= offset+height {
		offset = selected - height + 1
	}
	return clampRange(offset, 0, maxOffset)
}

func clampCursor(cursor int, value string) int {
	limit := len([]rune(value))
	if cursor < 0 {
		return 0
	}
	if cursor > limit {
		return limit
	}
	return cursor
}

func deleteBeforeCursor(value string, cursor int) (string, int) {
	runes := []rune(value)
	if cursor <= 0 || cursor > len(runes) {
		return value, clampCursor(cursor, value)
	}
	next := append(append([]rune{}, runes[:cursor-1]...), runes[cursor:]...)
	return string(next), cursor - 1
}

func deleteAtCursor(value string, cursor int) (string, int) {
	runes := []rune(value)
	if cursor < 0 || cursor >= len(runes) {
		return value, clampCursor(cursor, value)
	}
	next := append(append([]rune{}, runes[:cursor]...), runes[cursor+1:]...)
	return string(next), cursor
}

func insertAtCursor(value string, cursor int, text string) (string, int) {
	runes := []rune(value)
	insert := []rune(text)
	cursor = clampCursor(cursor, value)
	next := append(append(append([]rune{}, runes[:cursor]...), insert...), runes[cursor:]...)
	return string(next), cursor + len(insert)
}

func withCursor(value string, cursor int, cursorToken string) string {
	runes := []rune(value)
	cursor = clampCursor(cursor, value)
	if cursor == len(runes) {
		return value + cursorToken
	}
	return string(runes[:cursor]) + cursorToken + string(runes[cursor:])
}

func maxInt(a int, b int) int {
	if a > b {
		return a
	}
	return b
}

func minInt(a int, b int) int {
	if a < b {
		return a
	}
	return b
}
