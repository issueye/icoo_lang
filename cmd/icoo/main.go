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
	if len(os.Args) < 2 {
		runRepl()
		return
	}

	switch os.Args[1] {
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
	if len(args) != 1 {
		return fmt.Errorf("usage: icoo run <file|dir>")
	}
	return runProjectPath(args[0])
}

func printUsage() {
	fmt.Println("Icoo CLI")
	fmt.Println()
	fmt.Println("Usage:")
	fmt.Println("  icoo                                start REPL")
	fmt.Println("  icoo repl                           start REPL")
	fmt.Println("  icoo init [dir] [--entry path] [--entry-fn name]")
	fmt.Println("                                      initialize project")
	fmt.Println("  icoo check <file|dir>               check source file or project")
	fmt.Println("  icoo run <file|dir>                 run source file or project")
}

func runRepl() {
	fmt.Println("Icoo REPL")
	fmt.Println("Enter expressions or statements. Type :q to quit.")
	fmt.Println()

	rt := api.NewRuntime()
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
