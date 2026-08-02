// Package main is the entry point for the api-tester CLI tool.
// It provides a simple HTTP client for testing APIs with support for various
// HTTP methods, headers, request bodies, and response formatting.
package main

import (
	"errors"
	"fmt"
	"os"

	"ilo-pana/cmd/ilo-pana/commands"
	"ilo-pana/internal/client"
	"ilo-pana/internal/config"
)

func main() {
	// Check if this is a subcommand
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "session":
			commands.SessionCommand(os.Args[2:])
			return
		case "help", "--help", "-h":
			printHelp()
			return
		}
	}
	
	// Parse command-line flags and configuration
	cfg, err := config.ParseWith(os.Args[1:], os.Stderr)
	if err != nil {
		// Flag errors were already reported by the flag package together
		// with the usage text; only add our own message for other errors.
		if errors.Is(err, config.ErrUsage) {
			os.Exit(2)
		}
		fmt.Fprintf(os.Stderr, "Error: %v\n\n", err)
		config.Usage(os.Stderr)
		os.Exit(1)
	}

	// Create and execute the HTTP client
	httpClient, err := client.New(cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	
	if err := httpClient.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func printHelp() {
	fmt.Println("ilo-pana - A simple and powerful HTTP API testing tool")
	fmt.Println()
	fmt.Println("Usage:")
	fmt.Println("  ilo-pana [OPTIONS] URL        Make an HTTP request")
	fmt.Println("  ilo-pana session <command>    Manage sessions")
	fmt.Println()
	fmt.Println("Run 'ilo-pana --help' for HTTP options")
	fmt.Println("Run 'ilo-pana session --help' for session commands")
}
