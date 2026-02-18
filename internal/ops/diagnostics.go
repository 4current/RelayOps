package ops

import (
	"context"
	"fmt"
	"os/exec"
	"regexp"
	"strings"

	"github.com/4current/relayops/internal/core"
	"github.com/4current/relayops/internal/runtime"
	"github.com/4current/relayops/internal/store"
	"github.com/4current/relayops/internal/transport/pat"
)

// InitRuntime ensures ~/.relayops exists and the DB schema is ready (SQLite open + migrate).
func InitRuntime(ctx context.Context) error {
	_, err := runtime.EnsureAppDir()
	if err != nil {
		return err
	}
	st, err := store.Open(ctx)
	if err != nil {
		return err
	}
	return st.Close()
}

// Doctor validates core invariants needed to run RelayOps.
func Doctor(ctx context.Context) error {
	fmt.Println("Running diagnostics...")

	// Includes SQLite open+migrate
	if err := InitRuntime(ctx); err != nil {
		return err
	}
	fmt.Println("✔ runtime + sqlite: OK")

	// message model
	testMsg := core.NewMessage("Test Subject", "Test Body")
	if testMsg == nil || testMsg.ID == "" {
		return fmt.Errorf("doctor: NewMessage produced empty message or ID")
	}
	if testMsg.Status == "" {
		return fmt.Errorf("doctor: NewMessage produced empty status")
	}
	fmt.Printf("✔ message model OK (ID: %s)\n", testMsg.ID)

	// ---- PAT checks ----

	// 1) pat binary present?
	if p, err := exec.LookPath("pat"); err != nil {
		fmt.Printf("⚠ pat binary: not found in PATH (install PAT or fix PATH)\n")
	} else {
		fmt.Printf("✔ pat binary: %s\n", p)
	}

	// 2) pat config present + readable?
	cfg, cfgPath, err := pat.LoadConfig()
	if err != nil {
		fmt.Printf("⚠ pat config: %v\n", err)
	} else {
		fmt.Printf("✔ pat config: %s\n", cfgPath)
		if err := validateCallsign(cfg.MyCall); err != nil {
			fmt.Printf("⚠ pat mycall: %v\n", err)
		} else {
			fmt.Printf("✔ pat mycall: %s\n", cfg.MyCall)
		}
	}

	fmt.Println("Diagnostics complete.")
	return nil
}

var callsignRe = regexp.MustCompile(`^[A-Z0-9/]{3,16}$`)

func validateCallsign(s string) error {
	cs := strings.ToUpper(strings.TrimSpace(s))
	if cs == "" {
		return fmt.Errorf("empty callsign")
	}
	if !callsignRe.MatchString(cs) {
		return fmt.Errorf("invalid callsign %q (expected 3-16 of A-Z, 0-9, /)", s)
	}
	return nil
}
