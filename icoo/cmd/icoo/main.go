package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"icoo_lang/internal/runtime"
	"icoo_lang/pkg/api"
)

func main() {
	args := os.Args[1:]
	if len(args) > 0 && args[0] == "--icoo-cli" {
		os.Args = append([]string{os.Args[0]}, args[1:]...)
	} else {
		ran, err := runEmbeddedBundleIfPresent()
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		if ran {
			return
		}
	}

	if len(os.Args) < 2 {
		runRepl()
		return
	}

	switch os.Args[1] {
	case "build":
		if err := runBuild(os.Args[2:]); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	case "extract":
		if err := runExtract(os.Args[2:]); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	case "inspect":
		if err := runInspect(os.Args[2:]); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	case "bundle":
		if err := runBundle(os.Args[2:]); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	case "package":
		if err := runPackage(os.Args[2:]); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	case "check":
		if err := runCheck(os.Args[2:]); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	case "run":
		if err := runRun(os.Args[2:]); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	case "init":
		if err := runInit(os.Args[2:]); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	case "init-pkg":
		if err := runInitPackage(os.Args[2:]); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	case "init-subpkg":
		if err := runInitSubpackage(os.Args[2:]); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	case "repl":
		runRepl()
	case "help", "--help", "-h":
		printUsage()
	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s\n\n", os.Args[1])
		printUsage()
		os.Exit(1)
	}
}

func runCheck(args []string) error {
	if len(args) != 1 {
		return fmt.Errorf("usage: icoo check <file|dir>")
	}
	return runCheckPath(args[0])
}

func runRun(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: icoo run <file|dir> [-- script args...]")
	}
	split := len(args)
	for i, arg := range args {
		if arg == "--" {
			split = i
			break
		}
	}
	if split != 1 {
		return fmt.Errorf("usage: icoo run <file|dir> [-- script args...]")
	}
	scriptArgs := []string{}
	if split < len(args) {
		scriptArgs = args[split+1:]
	}
	return runProjectPath(args[0], scriptArgs)
}

func printUsage() {
	fmt.Println("Icoo CLI")
	fmt.Println()
	fmt.Println("Usage:")
	fmt.Println("  icoo                                                   start REPL")
	fmt.Println("  icoo repl                                              start REPL")
	fmt.Println("  icoo init [dir] [--entry path] [--entry-fn name] [--root-alias name]")
	fmt.Println("                                                         initialize project")
	fmt.Println("  icoo init-pkg [dir] [--name value] [--version value]   initialize package scaffold with pkg.toml")
	fmt.Println("  icoo init-subpkg <dir> --parent value                  initialize pkg/<name> style subpackage scaffold")
	fmt.Println("  icoo bundle <file|dir> [output]                        bundle source into one .icb file")
	fmt.Println("  icoo package <file|dir> [output] [--export path]       package source into one reusable .icpkg file")
	fmt.Println("  icoo build <file|dir> [output] [--metadata file]       build a standalone executable with embedded bundle")
	fmt.Println("  icoo extract <bundle|executable> [output]              extract embedded bundle to an .icb file")
	fmt.Println("  icoo inspect <bundle|executable>                       inspect bundled modules and entry metadata")
	fmt.Println("  icoo check <file|dir>                                  check source file or project")
	fmt.Println("  icoo run <file|dir> [-- script args...]               run source file or project")
}

func runRepl() {
	fmt.Println("Icoo REPL")
	fmt.Println("Enter expressions or statements. Type :q to quit.")
	fmt.Println()

	rt := api.NewRuntime()
	defer func() {
		_ = rt.Close()
	}()
	scanner := bufio.NewScanner(os.Stdin)

	for {
		fmt.Print("> ")
		if !scanner.Scan() {
			break
		}

		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		if line == ":q" || line == ":quit" {
			fmt.Println("bye.")
			break
		}

		result, err := rt.RunReplLine(line)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			continue
		}
		if result != nil && result.Kind() != runtime.NullKind {
			fmt.Println(result.String())
		}
	}
}
