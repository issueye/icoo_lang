const cp = require("child_process");
const fs = require("fs");
const path = require("path");

const rootDir = path.resolve(__dirname, "..");
const packageJsonPath = path.join(rootDir, "package.json");
const packageJson = JSON.parse(fs.readFileSync(packageJsonPath, "utf8"));
const validateOnly = process.argv.includes("--validate");

const requiredFiles = [
  "package.json",
  "extension.js",
  "language-configuration.json",
  "README.md",
  ".vscodeignore",
  "syntaxes/icoo.tmLanguage.json",
  "snippets/icoo.code-snippets"
];

function main() {
  validateProjectFiles();
  console.log("Icoo extension validation passed.");

  if (validateOnly) {
    return;
  }

  const distDir = path.join(rootDir, "dist");
  fs.mkdirSync(distDir, { recursive: true });

  const outputFile = path.join(distDir, `${packageJson.name}-${packageJson.version}.vsix`);
  if (fs.existsSync(outputFile)) {
    fs.rmSync(outputFile);
  }

  console.log(`Packaging VS Code extension to ${outputFile}`);
  runCommand(
    getNpxCommand(),
    ["--yes", "@vscode/vsce", "package", "--allow-missing-repository", "--out", outputFile],
    rootDir
  );
  console.log(`Build complete: ${outputFile}`);
}

function validateProjectFiles() {
  for (const relativePath of requiredFiles) {
    const fullPath = path.join(rootDir, relativePath);
    if (!fs.existsSync(fullPath)) {
      throw new Error(`Required file is missing: ${relativePath}`);
    }
  }

  parseJson("package.json");
  parseJson("language-configuration.json");
  parseJson("syntaxes/icoo.tmLanguage.json");
  parseJson("snippets/icoo.code-snippets");

  if (!packageJson.name || !packageJson.version) {
    throw new Error("package.json must define both name and version.");
  }
}

function parseJson(relativePath) {
  const filePath = path.join(rootDir, relativePath);
  JSON.parse(fs.readFileSync(filePath, "utf8"));
}

function getNpxCommand() {
  return process.platform === "win32" ? "npx.cmd" : "npx";
}

function runCommand(command, args, cwd) {
  const result = cp.spawnSync(command, args, {
    cwd,
    stdio: "inherit",
    shell: process.platform === "win32"
  });

  if (result.error) {
    throw result.error;
  }
  if (typeof result.status === "number" && result.status !== 0) {
    process.exit(result.status);
  }
}

try {
  main();
} catch (error) {
  console.error(error instanceof Error ? error.message : String(error));
  process.exit(1);
}
