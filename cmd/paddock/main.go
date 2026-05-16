package main

import (
	"fmt"
	"os"
)

const version = "0.0.1-dev"

func main() {
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "--version", "-v":
			fmt.Println("paddock", version)
			return
		}
	}
	fmt.Println("paddock — multi-Claude control plane")
	fmt.Println("Subcommands land in Round 1. Run with --version for version info.")
}
