const cp = require("child_process");
const fs = require("fs");
const path = require("path");
const vscode = require("vscode");

const DIAGNOSTIC_SOURCE = "icoo";
const CHECK_DEBOUNCE_MS = 200;

let diagnostics;
let output;
const pendingChecks = new Map();

function activate(context) {
  diagnostics = vscode.languages.createDiagnosticCollection(DIAGNOSTIC_SOURCE);
  output = vscode.window.createOutputChannel("Icoo");

  context.subscriptions.push(diagnostics, output);
  context.subscriptions.push(
    vscode.commands.registerCommand("icoo.checkCurrentFile", () => checkCurrentFile(true)),
    vscode.commands.registerCommand("icoo.runCurrentFile", runCurrentFile),
    vscode.workspace.onDidOpenTextDocument((document) => {
      scheduleDiagnostics(document, true);
    }),
    vscode.workspace.onDidSaveTextDocument((document) => {
      scheduleDiagnostics(document, true);
    }),
    vscode.workspace.onDidCloseTextDocument((document) => {
      cancelScheduledDiagnostics(document.uri);
      diagnostics.delete(document.uri);
    })
  );

  for (const document of vscode.workspace.textDocuments) {
    scheduleDiagnostics(document, false);
  }
}

function deactivate() {
  for (const timer of pendingChecks.values()) {
    clearTimeout(timer);
  }
  pendingChecks.clear();
}

function scheduleDiagnostics(document, immediate) {
  if (!shouldHandleDocument(document) || !diagnosticsEnabled()) {
    return;
  }

  cancelScheduledDiagnostics(document.uri);
  const delay = immediate ? 0 : CHECK_DEBOUNCE_MS;
  const timer = setTimeout(() => {
    pendingChecks.delete(document.uri.toString());
    void updateDiagnostics(document);
  }, delay);
  pendingChecks.set(document.uri.toString(), timer);
}

function cancelScheduledDiagnostics(uri) {
  const key = uri.toString();
  const timer = pendingChecks.get(key);
  if (timer) {
    clearTimeout(timer);
    pendingChecks.delete(key);
  }
}

async function updateDiagnostics(document) {
  if (!shouldHandleDocument(document) || document.isUntitled || document.isDirty || !diagnosticsEnabled()) {
    diagnostics.delete(document.uri);
    return;
  }

  const result = await invokeIcoo(document, "check");
  if (result.ok) {
    diagnostics.delete(document.uri);
    return;
  }

  const parsed = parseDiagnostics(document, result.output);
  diagnostics.set(document.uri, parsed);
}

async function checkCurrentFile(showResult) {
  const document = getActiveIcooDocument();
  if (!document) {
    vscode.window.showWarningMessage("Open an Icoo (.ic) file first.");
    return;
  }

  if (document.isDirty) {
    const saved = await document.save();
    if (!saved) {
      vscode.window.showWarningMessage("Save the file before running Icoo check.");
      return;
    }
  }

  const result = await invokeIcoo(document, "check");
  const parsed = result.ok ? [] : parseDiagnostics(document, result.output);
  diagnostics.set(document.uri, parsed);

  if (result.ok && showResult) {
    vscode.window.showInformationMessage(`Icoo check passed: ${path.basename(document.uri.fsPath)}`);
  } else if (!result.ok && showResult) {
    output.show(true);
    vscode.window.showErrorMessage(`Icoo check failed: ${path.basename(document.uri.fsPath)}`);
  }
}

async function runCurrentFile() {
  const document = getActiveIcooDocument();
  if (!document) {
    vscode.window.showWarningMessage("Open an Icoo (.ic) file first.");
    return;
  }

  if (document.isDirty) {
    const saved = await document.save();
    if (!saved) {
      vscode.window.showWarningMessage("Save the file before running it.");
      return;
    }
  }

  const cli = resolveCli(document);
  const workspaceFolder = vscode.workspace.getWorkspaceFolder(document.uri);
  const execution = new vscode.ProcessExecution(
    cli.command,
    [...cli.args, "run", document.uri.fsPath],
    cli.cwd ? { cwd: cli.cwd } : undefined
  );
  const task = new vscode.Task(
    { type: "process", command: cli.command },
    workspaceFolder || vscode.TaskScope.Workspace,
    `Run ${path.basename(document.uri.fsPath)}`,
    "icoo",
    execution
  );
  task.presentationOptions = {
    clear: false,
    reveal: vscode.TaskRevealKind.Always,
    focus: true
  };

  await vscode.tasks.executeTask(task);
}

function getActiveIcooDocument() {
  const editor = vscode.window.activeTextEditor;
  if (!editor || !shouldHandleDocument(editor.document)) {
    return null;
  }
  return editor.document;
}

function shouldHandleDocument(document) {
  return Boolean(document && document.uri.scheme === "file" && document.languageId === "icoo");
}

function diagnosticsEnabled() {
  return vscode.workspace.getConfiguration("icoo").get("diagnostics.enabled", true);
}

async function invokeIcoo(document, subcommand) {
  const cli = resolveCli(document);
  const args = [...cli.args, subcommand, document.uri.fsPath];
  const label = `${cli.command} ${args.join(" ")}`;

  output.appendLine(`> ${label}`);
  if (cli.cwd) {
    output.appendLine(`cwd: ${cli.cwd}`);
  }

  try {
    const result = await runProcess(cli.command, args, cli.cwd);
    if (result.output) {
      output.appendLine(result.output.trimEnd());
    }
    return result;
  } catch (error) {
    const message = error instanceof Error ? error.message : String(error);
    output.appendLine(message);
    return { ok: false, output: message };
  }
}

function resolveCli(document) {
  const configuration = vscode.workspace.getConfiguration("icoo", document.uri);
  const configuredCommand = configuration.get("cli.command", "").trim();
  const configuredArgs = normalizeStringArray(configuration.get("cli.args", []));
  const workspaceFolder = vscode.workspace.getWorkspaceFolder(document.uri);
  const configuredCwd = resolveConfiguredCwd(configuration.get("cli.cwd", ""), workspaceFolder);

  if (configuredCommand) {
    return {
      command: configuredCommand,
      args: configuredArgs,
      cwd: configuredCwd
    };
  }

  if (workspaceFolder) {
    const workspaceRoot = workspaceFolder.uri.fsPath;
    const detectedCli = detectWorkspaceCli(workspaceRoot);
    if (detectedCli) {
      return detectedCli;
    }
  }

  return {
    command: "icoo",
    args: [],
    cwd: configuredCwd
  };
}

function detectWorkspaceCli(workspaceRoot) {
  for (const candidateRoot of [workspaceRoot, path.join(workspaceRoot, "icoo")]) {
    if (fs.existsSync(path.join(candidateRoot, "go.mod")) && fs.existsSync(path.join(candidateRoot, "cmd", "icoo", "main.go"))) {
      return {
        command: "go",
        args: ["run", "./cmd/icoo"],
        cwd: candidateRoot
      };
    }

    const localCli = path.join(candidateRoot, process.platform === "win32" ? "icoo.exe" : "icoo");
    if (fs.existsSync(localCli)) {
      return {
        command: localCli,
        args: [],
        cwd: candidateRoot
      };
    }
  }

  return null;
}

function resolveConfiguredCwd(rawValue, workspaceFolder) {
  const value = String(rawValue || "").trim();
  if (!value) {
    return workspaceFolder ? workspaceFolder.uri.fsPath : undefined;
  }

  const expanded = workspaceFolder ? value.replaceAll("${workspaceFolder}", workspaceFolder.uri.fsPath) : value;
  return path.isAbsolute(expanded) ? expanded : workspaceFolder ? path.join(workspaceFolder.uri.fsPath, expanded) : expanded;
}

function normalizeStringArray(value) {
  if (!Array.isArray(value)) {
    return [];
  }
  return value.filter((item) => typeof item === "string");
}

function runProcess(command, args, cwd) {
  return new Promise((resolve, reject) => {
    const child = cp.spawn(command, args, {
      cwd,
      windowsHide: true
    });

    let stdout = "";
    let stderr = "";

    child.stdout.on("data", (chunk) => {
      stdout += chunk.toString();
    });
    child.stderr.on("data", (chunk) => {
      stderr += chunk.toString();
    });
    child.on("error", (error) => {
      reject(new Error(`Failed to start Icoo CLI "${command}": ${error.message}`));
    });
    child.on("close", (code) => {
      const outputText = [stdout, stderr].filter(Boolean).join("\n").trim();
      resolve({
        ok: code === 0,
        output: outputText
      });
    });
  });
}

function parseDiagnostics(document, outputText) {
  const diagnosticsList = [];
  const lines = String(outputText || "")
    .split(/\r?\n/)
    .map((line) => line.trim())
    .filter(Boolean);

  for (const line of lines) {
    const match = /^(\d+):(\d+):\s+(.*)$/.exec(line);
    if (!match) {
      continue;
    }

    const lineIndex = Math.max(0, Number(match[1]) - 1);
    const columnIndex = Math.max(0, Number(match[2]) - 1);
    const message = match[3];
    const range = createRange(document, lineIndex, columnIndex);
    diagnosticsList.push(new vscode.Diagnostic(range, message, vscode.DiagnosticSeverity.Error));
  }

  if (diagnosticsList.length > 0) {
    return diagnosticsList;
  }

  const fallbackMessage = lines.length > 0 ? lines.join("\n") : "Icoo check failed.";
  const fallbackRange = new vscode.Range(new vscode.Position(0, 0), new vscode.Position(0, 0));
  return [new vscode.Diagnostic(fallbackRange, fallbackMessage, vscode.DiagnosticSeverity.Error)];
}

function createRange(document, lineIndex, columnIndex) {
  const safeLine = Math.min(lineIndex, Math.max(0, document.lineCount - 1));
  const textLine = document.lineAt(safeLine);
  const start = new vscode.Position(safeLine, Math.min(columnIndex, textLine.text.length));
  const endColumn = start.character < textLine.text.length ? start.character + 1 : start.character;
  const end = new vscode.Position(safeLine, endColumn);
  return new vscode.Range(start, end);
}

module.exports = {
  activate,
  deactivate
};
