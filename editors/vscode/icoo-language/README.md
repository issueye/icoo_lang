# Icoo VS Code Extension

This extension adds first-party editor support for the Icoo language:

- `.ic` language registration
- TextMate syntax highlighting
- comment, bracket, and auto-close rules
- snippets for common Icoo constructs
- `Icoo: Check Current File`
- `Icoo: Run Current File`
- automatic diagnostics on open/save through `icoo check`

## Workspace-friendly CLI detection

By default the extension will:

1. use `go run ./cmd/icoo` when the current workspace looks like the Icoo source repo
2. otherwise fall back to `icoo` from `PATH`

You can override that behavior with:

- `icoo.cli.command`
- `icoo.cli.args`
- `icoo.cli.cwd`

Example settings:

```json
{
  "icoo.cli.command": "go",
  "icoo.cli.args": ["run", "./cmd/icoo"],
  "icoo.cli.cwd": "${workspaceFolder}"
}
```

## Local development

Open this folder in VS Code and press `F5` to launch an Extension Development Host:

`E:\\codes\\icoo_lang\\editors\\vscode\\icoo-language`

Then open any `.ic` file and run the commands from the Command Palette.

## Build

Validate the extension files:

```powershell
npm run check
```

Build the installable `.vsix` package:

```powershell
npm run build
```

The packaged extension will be written to `dist/`.

Windows users can also run:

```powershell
.\build.ps1
```

The build script packages `dist/icoo-language-<version>.vsix`.
