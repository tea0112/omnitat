package main

import (
	"fmt"
	"os"

	"github.com/tea0112/omnitat/services/go/iam/identity/internal/config"
)

func runCommand() {
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load config: %v\n", err)
		os.Exit(1)
	}

	if len(os.Args) < 2 {
		fmt.Println("Usage: identity [server|migrate|seed]")
		os.Exit(1)
	}

	command := os.Args[1]

	switch command {
	case "serve":
		err = runServer(cfg)
	case "migrate":
		err = runMigration(cfg)
	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n", command)
		os.Exit(1)
	}

	if err != nil {
		fmt.Fprintf(os.Stderr, "Command failed: %v\n", err)
		os.Exit(1)
	}
}
