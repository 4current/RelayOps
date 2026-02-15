package ops

import (
	"context"
	"fmt"

	"github.com/4current/relayops/internal/core"
	"github.com/4current/relayops/internal/runtime"
	"github.com/4current/relayops/internal/store"
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
	// Includes SQLite open+migrate
	if err := InitRuntime(ctx); err != nil {
		return err
	}

	// Core model sanity check (mirrors CLI doctor)
	m := core.NewMessage("Test Subject", "Test Body")
	if m == nil || m.ID == "" {
		return fmt.Errorf("doctor: NewMessage produced empty message or ID")
	}
	if m.Status == "" {
		return fmt.Errorf("doctor: NewMessage produced empty status")
	}
	return nil
}
