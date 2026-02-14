package main

import (
	"fmt"
	"os"

	"github.com/4current/relayops/internal/core"
)

var (
	version   = "0.0.2-alpha"
	commit    = "none"
	buildDate = "unknown"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		return
	}

	switch os.Args[1] {

	case "version":
		fmt.Printf("RelayOps %s\nCommit: %s\nBuilt: %s\n", version, commit, buildDate)

	case "doctor":
		runDoctor()

	default:
		fmt.Printf("Unknown command: %s\n\n", os.Args[1])
		printUsage()
	}
}

func printUsage() {
	fmt.Println("RelayOps - Radio Messaging Operations Engine")
	fmt.Println("")
	fmt.Println("Usage:")
	fmt.Println("  relayops version      Show version")
	fmt.Println("  relayops doctor       Run system diagnostics")
	fmt.Println("")
}

func runDoctor() {
	fmt.Println("Running diagnostics...")

	// Placeholder checks — expand later
	fmt.Println("✔ Core package loaded")

	testMsg := core.NewMessage("Test Subject", "Test Body")
	fmt.Printf("✔ Message model OK (ID: %s)\n", testMsg.ID)

	fmt.Println("Diagnostics complete.")
}
