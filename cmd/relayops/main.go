package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/4current/relayops/internal/core"
	"github.com/4current/relayops/internal/runtime"
	"github.com/4current/relayops/internal/store"
)

var (
	version   = "dev"
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

	case "init":
		runInit()

	case "compose":
		runCompose(os.Args[2:])

	case "list":
		runList(os.Args[2:])

	default:
		fmt.Printf("Unknown command: %s\n\n", os.Args[1])
		printUsage()
	}
}

func printUsage() {
	fmt.Println("RelayOps - Radio Messaging Operations Engine")
	fmt.Println("")
	fmt.Println("Usage:")
	fmt.Println("  relayops version")
	fmt.Println("  relayops doctor")
	fmt.Println("  relayops init")
	fmt.Println("  relayops compose -s \"subject\" -b \"body\" [-t tag1,tag2]")
	fmt.Println("  relayops list [-n 25]")
	fmt.Println("")
}

func runDoctor() {
	fmt.Println("Running diagnostics...")

	dir, err := runtime.EnsureAppDir()
	if err != nil {
		fmt.Printf("✘ runtime dir: %v\n", err)
		return
	}
	fmt.Printf("✔ runtime dir: %s\n", dir)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	st, err := store.Open(ctx)
	if err != nil {
		fmt.Printf("✘ store open: %v\n", err)
		return
	}
	_ = st.Close()
	fmt.Println("✔ sqlite store: OK")

	testMsg := core.NewMessage("Test Subject", "Test Body")
	fmt.Printf("✔ message model OK (ID: %s)\n", testMsg.ID)

	fmt.Println("Diagnostics complete.")
}

func runInit() {
	dir, err := runtime.EnsureAppDir()
	if err != nil {
		fmt.Printf("Init failed: %v\n", err)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	st, err := store.Open(ctx)
	if err != nil {
		fmt.Printf("Init failed (store): %v\n", err)
		return
	}
	_ = st.Close()

	fmt.Printf("Initialized RelayOps runtime at: %s\n", dir)
}

func runCompose(args []string) {
	fs := flag.NewFlagSet("compose", flag.ContinueOnError)
	subject := fs.String("s", "", "subject")
	body := fs.String("b", "", "body")
	tagCSV := fs.String("t", "", "comma-separated tags")
	_ = fs.Parse(args)

	if strings.TrimSpace(*subject) == "" || strings.TrimSpace(*body) == "" {
		fmt.Println("compose requires -s (subject) and -b (body)")
		fmt.Println("Example: relayops compose -s \"Winlink Wednesday\" -b \"Check-in\" -t winlink_wednesday")
		return
	}

	msg := core.NewMessage(*subject, *body)
	if strings.TrimSpace(*tagCSV) != "" {
		for _, t := range strings.Split(*tagCSV, ",") {
			t = strings.TrimSpace(t)
			if t != "" {
				msg.Tags = append(msg.Tags, t)
			}
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	st, err := store.Open(ctx)
	if err != nil {
		fmt.Printf("store open failed: %v\n", err)
		return
	}
	defer func() { _ = st.Close() }()

	if err := st.SaveMessage(ctx, msg); err != nil {
		fmt.Printf("save failed: %v\n", err)
		return
	}

	fmt.Printf("Saved message: %s\n", msg.ID)
}

func runList(args []string) {
	fs := flag.NewFlagSet("list", flag.ContinueOnError)
	n := fs.Int("n", 25, "number of messages")
	_ = fs.Parse(args)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	st, err := store.Open(ctx)
	if err != nil {
		fmt.Printf("store open failed: %v\n", err)
		return
	}
	defer func() { _ = st.Close() }()

	msgs, err := st.ListMessages(ctx, *n)
	if err != nil {
		fmt.Printf("list failed: %v\n", err)
		return
	}

	if len(msgs) == 0 {
		fmt.Println("(no messages)")
		return
	}

	for _, m := range msgs {
		ts := m.CreatedAt.Local().Format("2006-01-02 15:04:05")
		if len(m.Tags) > 0 {
			fmt.Printf("%s  %s  [%s]\n    %s\n", ts, m.ID, strings.Join(m.Tags, ","), m.Subject)
		} else {
			fmt.Printf("%s  %s\n    %s\n", ts, m.ID, m.Subject)
		}
	}
}
