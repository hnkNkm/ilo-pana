// Package commands implements subcommands for the ilo-pana CLI
package commands

import (
	"flag"
	"fmt"
	"os"
	
	"ilo-pana/internal/session"
)

// SessionCommand handles session subcommands
func SessionCommand(args []string) {
	if len(args) == 0 {
		sessionHelp()
		return
	}
	
	switch args[0] {
	case "list":
		sessionList(args[1:])
	case "show":
		sessionShow(args[1:])
	case "clear":
		sessionClear(args[1:])
	case "help", "--help", "-h":
		sessionHelp()
	default:
		fmt.Fprintf(os.Stderr, "Unknown session command: %s\n", args[0])
		sessionHelp()
		os.Exit(1)
	}
}

func sessionHelp() {
	fmt.Println("Session management commands")
	fmt.Println()
	fmt.Println("Usage:")
	fmt.Println("  ilo-pana session list              List all sessions")
	fmt.Println("  ilo-pana session show <name>       Show session details")
	fmt.Println("  ilo-pana session clear <name>      Clear a session")
	fmt.Println()
	fmt.Println("Options:")
	fmt.Println("  -v, --verbose    Show full session details (unmask values)")
}

func sessionList(args []string) {
	fs := flag.NewFlagSet("session list", flag.ExitOnError)
	fs.Parse(args)
	
	storage := session.NewFileStorage("")
	sessions, err := storage.List()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error listing sessions: %v\n", err)
		os.Exit(1)
	}
	
	if len(sessions) == 0 {
		fmt.Println("No sessions found")
		return
	}
	
	fmt.Println("Available sessions:")
	for _, name := range sessions {
		// Load session to get basic info
		sess, err := storage.Load(name)
		if err != nil {
			fmt.Printf("  %s (error loading)\n", name)
			continue
		}
		
		cookieCount := len(sess.Cookies)
		headerCount := len(sess.Headers)
		fmt.Printf("  %s (cookies: %d, headers: %d, updated: %s)\n", 
			name, cookieCount, headerCount, sess.Updated.Format("2006-01-02 15:04"))
	}
}

func sessionShow(args []string) {
	fs := flag.NewFlagSet("session show", flag.ExitOnError)
	verbose := fs.Bool("v", false, "Show full details (unmask values)")
	verboseLong := fs.Bool("verbose", false, "Show full details (unmask values)")
	fs.Parse(args)
	
	if fs.NArg() < 1 {
		fmt.Fprintf(os.Stderr, "Error: session name required\n")
		fmt.Fprintf(os.Stderr, "Usage: ilo-pana session show <name>\n")
		os.Exit(1)
	}
	
	name := fs.Arg(0)
	isVerbose := *verbose || *verboseLong
	
	storage := session.NewFileStorage("")
	sess, err := storage.Load(name)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading session %q: %v\n", name, err)
		os.Exit(1)
	}
	
	session.ShowSession(os.Stdout, sess, isVerbose)
}

func sessionClear(args []string) {
	fs := flag.NewFlagSet("session clear", flag.ExitOnError)
	fs.Parse(args)
	
	if fs.NArg() < 1 {
		fmt.Fprintf(os.Stderr, "Error: session name required\n")
		fmt.Fprintf(os.Stderr, "Usage: ilo-pana session clear <name>\n")
		os.Exit(1)
	}
	
	name := fs.Arg(0)
	
	// Confirm deletion
	fmt.Printf("Are you sure you want to clear session %q? (y/N): ", name)
	var response string
	fmt.Scanln(&response)
	
	if response != "y" && response != "Y" {
		fmt.Println("Cancelled")
		return
	}
	
	storage := session.NewFileStorage("")
	if err := storage.Delete(name); err != nil {
		fmt.Fprintf(os.Stderr, "Error clearing session: %v\n", err)
		os.Exit(1)
	}
	
	fmt.Printf("Session %q cleared\n", name)
}