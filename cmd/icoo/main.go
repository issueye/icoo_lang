package main

import (
	"fmt"
	"os"
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

	fmt.Printf("check not implemented yet: %s\n", args[0])
}

func runRun(args []string) {
	if len(args) != 1 {
		fmt.Fprintln(os.Stderr, "usage: icoo run <file>")
		os.Exit(1)
	}

	fmt.Printf("run not implemented yet: %s\n", args[0])
}

func printUsage() {
	fmt.Println("Icoo CLI")
	fmt.Println()
	fmt.Println("Usage:")
	fmt.Println("  icoo check <file>")
	fmt.Println("  icoo run <file>")
}
