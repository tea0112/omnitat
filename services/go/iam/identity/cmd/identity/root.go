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
	case "server":
		runServer(cfg)
	case "migration":
		runMigration(cfg)
	}
}