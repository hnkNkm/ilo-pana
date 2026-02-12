// Package main is the entry point for the api-tester CLI tool.
// It provides a simple HTTP client for testing APIs with support for various
// HTTP methods, headers, request bodies, and response formatting.
package main

import (
	"flag"
	"fmt"
	"os"

	"ilo-pana/internal/client"
	"ilo-pana/internal/config"
)

func main() {
	// Parse command-line flags and configuration
	cfg, err := config.Parse()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n\n", err)
		flag.Usage()
		os.Exit(1)
	}

	// Create and execute the HTTP client
	httpClient := client.New(cfg)
	if err := httpClient.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
