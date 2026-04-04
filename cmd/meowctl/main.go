package main

import (
	"fmt"
	"os"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("meowctl - MeowSight CLI")
		fmt.Println()
		fmt.Println("Usage: meowctl <command>")
		fmt.Println()
		fmt.Println("Commands:")
		fmt.Println("  version    Show version info")
		fmt.Println("  status     Check MeowSight service status")
		os.Exit(0)
	}

	switch os.Args[1] {
	case "version":
		fmt.Println("meowctl v0.1.0")
	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s\n", os.Args[1])
		os.Exit(1)
	}
}
