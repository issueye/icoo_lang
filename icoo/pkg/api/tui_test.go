package api

import (
	"testing"
)

func TestStdUITUIRenderAndStateHelpers(t *testing.T) {
	rt := NewRuntime()

	source := `
import std.core.string as str
import std.ui.tui as tui

fn main() {
  let menu = tui.listUpdate(
    tui.listState({
      items: ["Chat", "Logs", "Settings"],
      selected: 0
    }),
    {
      kind: "key",
      key: "down",
      text: ""
    }
  )

  if menu.selected != 1 {
    panic("expected listUpdate to move selection")
  }

  let prompt = tui.inputUpdate(
    tui.inputState({
      value: "hi",
      cursor: 2,
      focused: true
    }),
    {
      kind: "key",
      key: "!",
      text: "!"
    }
  )

  if prompt.value != "hi!" {
    panic("expected inputUpdate to append text")
  }

  let viewport = tui.viewportUpdate(
    tui.viewportState({
      content: "a\nb\nc\nd",
      height: 2,
      offset: 0
    }),
    {
      kind: "key",
      key: "down"
    }
  )
  if viewport.offset != 1 {
    panic("expected viewportUpdate to move offset")
  }

  let table = tui.tableUpdate(
    tui.tableState({
      columns: ["Name", "Role"],
      rows: [
        ["Alice", "Owner"],
        ["Bob", "Ops"],
        ["Cara", "Dev"]
      ],
      height: 2,
      selected: 0
    }),
    {
      kind: "key",
      key: "down"
    }
  )
  if table.selected != 1 {
    panic("expected tableUpdate to move selection")
  }

  let quitCommand = tui.quit()
  if quitCommand.type != "quit_command" {
    panic("expected quit command type")
  }

  let emitCommand = tui.emit({
    kind: "custom",
    text: "ping"
  })
  if emitCommand.type != "emit_command" {
    panic("expected emit command type")
  }

  let tickCommand = tui.tick(25, {
    kind: "refresh"
  })
  if tickCommand.type != "tick_command" {
    panic("expected tick command type")
  }

  let view = tui.vstack({
    gap: 1,
    children: [
      tui.status({
        text: "Connected",
        tone: "success"
      }),
      tui.box({
        title: "Prompt",
        border: "rounded",
        child: tui.input({
          state: prompt
        })
      }),
      tui.list({
        state: menu
      }),
      tui.viewport({
        state: viewport
      }),
      tui.table({
        state: table
      }),
      tui.progress({
        label: "Upload",
        value: 3,
        total: 5,
        width: 10
      }),
      tui.spinner({
        text: "Thinking"
      })
    ]
  })

  let rendered = tui.render(view)
  if str.indexOf(rendered, "Connected") < 0 {
    panic("expected rendered output to contain Connected")
  }
  if str.indexOf(rendered, "Prompt") < 0 {
    panic("expected rendered output to contain Prompt")
  }
  if str.indexOf(rendered, "hi!") < 0 {
    panic("expected rendered output to contain hi!")
  }
  if str.indexOf(rendered, "Logs") < 0 {
    panic("expected rendered output to contain Logs")
  }
  if str.indexOf(rendered, "b") < 0 || str.indexOf(rendered, "c") < 0 {
    panic("expected rendered output to contain viewport window")
  }
  if str.indexOf(rendered, "Bob | Ops") < 0 {
    panic("expected rendered output to contain table row")
  }
  if str.indexOf(rendered, "Upload") < 0 {
    panic("expected rendered output to contain progress label")
  }
  if str.indexOf(rendered, "Thinking") < 0 {
    panic("expected rendered output to contain Thinking")
  }
}

main()
`

	if _, err := rt.RunSource(source); err != nil {
		t.Fatalf("expected std.ui.tui script to succeed, got: %v", err)
	}
}
