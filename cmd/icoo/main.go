package main

import (
	"fmt"
	"os"

	"icoo_lang/pkg/api"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	switch os.Args[1] {
	case "check":
		runCheck(os.Args[2:])
	case "run":
		runRun(os.Args[2:])
	case "help", "--help", "-h":
		printUsage()
	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s\n\n", os.Args[1])
		printUsage()
		os.Exit(1)
	}
}

func runCheck(args []string) {
	if len(args) != 1 {
		fmt.Fprintln(os.Stderr, "usage: icoo check <file>")
		os.Exit(1)
	}

	rt := api.NewRuntime()
	errs := rt.CheckFile(args[0])
	if len(errs) > 0 {
		for _, err := range errs {
			fmt.Fprintln(os.Stderr, err)
		}
		os.Exit(1)
	}

	fmt.Printf("ok: %s\n", args[0])
}

func runRun(args []string) {
	if len(args) != 1 {
		fmt.Fprintln(os.Stderr, "usage: icoo run <file>")
		os.Exit(1)
	}

	rt := api.NewRuntime()
	_, err := rt.RunFile(args[0])
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println("Icoo CLI")
	fmt.Println()
	fmt.Println("Usage:")
	fmt.Println("  icoo check <file>")
	fmt.Println("  icoo run <file>")
}
