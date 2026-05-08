# Icoo Reusable Package Design

`icoo` now supports a reusable package format similar to a Java JAR for library distribution and application packaging.

## Package Types

- `.icb`
  Application bundle. Used for `icoo run`, `icoo build`, `icoo inspect`, and `icoo extract`.
- `.icpkg`
  Reusable package archive. Can be imported by other Icoo projects and can also carry a runnable entry.

Both archive types share the same archive model:

- `entry`
  The runnable module entry.
- `entry_function`
  Optional function invoked after the entry module is loaded.
- `export`
  The module exposed when another project imports the package.
- `modules`
  Bundled source files.
- `packages`
  Nested packaged dependencies that are carried into bundles and executables.

## Import Modes

Icoo supports two package import modes.

### 1. Local file package import

```icoo
import "./libs/greeter.icpkg" as greeter

if greeter.message() != "hello" {
  panic("unexpected package message")
}
```

### 2. Named package import

```icoo
import "pkg:acme/greeter" as greeter
```

Named packages are resolved from:

- `<projectRoot>/.icoo/packages/<name>.icpkg`
- `<projectRoot>/packages/<name>.icpkg`

For example, `pkg:acme/greeter` maps to:

- `.icoo/packages/acme/greeter.icpkg`
- `packages/acme/greeter.icpkg`

## Commands

Create a reusable package:

```powershell
icoo package .\demo .\dist\demo.icpkg --name acme/demo --version 1.0.0 --export src/lib/api.ic
```

Create an application bundle:

```powershell
icoo bundle .\demo .\dist\demo.icb
```

Build a standalone executable:

```powershell
icoo build .\demo .\dist\demo.exe
```

Run either a source project, bundle, or package:

```powershell
icoo run .\demo
icoo run .\dist\demo.icb
icoo run .\dist\demo.icpkg
```

## Recommended Layout

For reusable libraries:

```text
demo/
  project.toml
  src/
    main.ic
    lib/
      api.ic
```

- `src/main.ic`
  Optional runnable entry for local execution and app-style packaging.
- `src/lib/api.ic`
  Recommended exported surface for reuse via `--export`.

For named package storage inside a consumer project:

```text
consumer/
  project.toml
  .icoo/
    packages/
      acme/
        demo.icpkg
```
