package ops_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/4current/relayops/internal/core"
	"github.com/4current/relayops/internal/ops"
	"github.com/4current/relayops/internal/store"
	"github.com/4current/relayops/internal/transport/sim"
)

func withTempHome(t *testing.T) string {
	t.Helper()
	tmp := t.TempDir()

	// Redirect home so runtime.DBPath() goes into tmp/.relayops/relayops.db
	// macOS/Linux use HOME; Windows would use USERPROFILE (handle later if needed).
	if err := os.Setenv("HOME", tmp); err != nil {
		t.Fatalf("set HOME: %v", err)
	}
	return tmp
}

func openStore(t *testing.T) *store.Store {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	t.Cleanup(cancel)

	st, err := store.Open(ctx)
	if err != nil {
		t.Fatalf("store.Open: %v", err)
	}
	t.Cleanup(func() { _ = st.Close() })
	return st
}

func TestDraftToQueuedToSent(t *testing.T) {
	withTempHome(t)
	st := openStore(t)

	ctx := context.Background()

	// Draft
	msg := core.NewMessage("WW", "check-in")
	msg.Tags = []string{"winlink_wednesday"}
	msg.Meta.Transport.Allowed = []core.Mode{core.ModePacket}
	msg.Meta.Transport.Preferred = []core.Mode{core.ModePacket}
	msg.Meta.Session = core.SessionWinlink

	if err := st.SaveMessage(ctx, msg); err != nil {
		t.Fatalf("SaveMessage: %v", err)
	}

	// Queue
	n, err := st.QueueByTag(ctx, "winlink_wednesday")
	if err != nil {
		t.Fatalf("QueueByTag: %v", err)
	}
	if n != 1 {
		t.Fatalf("expected 1 queued, got %d", n)
	}

	// Send
	res, err := ops.SendQueued(ctx, st, "winlink_wednesday", 25, sim.New())
	if err != nil {
		t.Fatalf("SendQueued: %v", err)
	}
	if res.Sent != 1 || res.Failed != 0 {
		t.Fatalf("expected sent=1 failed=0, got sent=%d failed=%d", res.Sent, res.Failed)
	}

	// Verify DB state: sent
	got, err := st.ListByStatus(ctx, []core.MessageStatus{core.StatusSent}, 10)
	if err != nil {
		t.Fatalf("ListByStatus(sent): %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("expected 1 sent message, got %d", len(got))
	}
}

func TestQueuedToFailedOnRuleViolation(t *testing.T) {
	home := withTempHome(t)
	st := openStore(t)
	ctx := context.Background()

	msg := core.NewMessage("Bad", "should fail")
	msg.Tags = []string{"testfail"}
	msg.Meta.Transport.Allowed = []core.Mode{core.ModeTelnet} // telnet-only
	msg.Meta.Session = core.SessionRadioOnly                  // incompatible with telnet-only per sim sender rule

	if err := st.SaveMessage(ctx, msg); err != nil {
		t.Fatalf("SaveMessage: %v", err)
	}

	if _, err := st.QueueByTag(ctx, "testfail"); err != nil {
		t.Fatalf("QueueByTag: %v", err)
	}

	res, err := ops.SendQueued(ctx, st, "testfail", 25, sim.New())
	if err != nil {
		t.Fatalf("SendQueued: %v", err)
	}
	if res.Sent != 0 || res.Failed != 1 {
		t.Fatalf("expected sent=0 failed=1, got sent=%d failed=%d", res.Sent, res.Failed)
	}

	// Optionally ensure DB file is indeed in temp home
	db := filepath.Join(home, ".relayops", "relayops.db")
	if _, err := os.Stat(db); err != nil {
		t.Fatalf("expected db at %s, stat error: %v", db, err)
	}

	failed, err := st.ListByStatus(ctx, []core.MessageStatus{core.StatusFailed}, 10)
	if err != nil {
		t.Fatalf("ListByStatus(failed): %v", err)
	}
	if len(failed) != 1 {
		t.Fatalf("expected 1 failed message, got %d", len(failed))
	}
}
