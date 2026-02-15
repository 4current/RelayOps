package ops_test

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/4current/relayops/internal/core"
	"github.com/4current/relayops/internal/ops"
	"github.com/4current/relayops/internal/store"
	"github.com/4current/relayops/internal/transport/sim"
)

// setupTestStore isolates the RelayOps runtime (~/.relayops) into a temp HOME
// so tests do not touch your real ~/.relayops/relayops.db.
func setupTestStore(t *testing.T) (*store.Store, context.Context) {
	t.Helper()

	tmp := t.TempDir()

	// macOS/Linux
	_ = os.Setenv("HOME", tmp)
	// Windows (harmless elsewhere)
	_ = os.Setenv("USERPROFILE", tmp)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	t.Cleanup(cancel)

	st, err := store.Open(ctx)
	if err != nil {
		t.Fatalf("store.Open: %v", err)
	}
	t.Cleanup(func() { _ = st.Close() })

	return st, ctx
}

func TestDraftToQueuedToSent(t *testing.T) {
	st, ctx := setupTestStore(t)

	msg := core.NewMessage("OK", "body")
	msg.Tags = []string{"t_send"}

	if err := st.SaveMessage(ctx, msg); err != nil {
		t.Fatalf("SaveMessage: %v", err)
	}

	n, err := st.QueueByTag(ctx, "t_send")
	if err != nil {
		t.Fatalf("QueueByTag: %v", err)
	}
	if n != 1 {
		t.Fatalf("expected 1 queued, got %d", n)
	}

	res, err := ops.SendQueued(ctx, st, "t_send", 10, sim.New())
	if err != nil {
		t.Fatalf("SendQueued: %v", err)
	}
	if res.Sent != 1 || res.Failed != 0 {
		t.Fatalf("expected sent=1 failed=0, got sent=%d failed=%d", res.Sent, res.Failed)
	}

	sent, err := st.ListByStatus(ctx, []core.MessageStatus{core.StatusSent}, 10)
	if err != nil {
		t.Fatalf("ListByStatus(sent): %v", err)
	}
	if len(sent) != 1 {
		t.Fatalf("expected 1 sent message, got %d", len(sent))
	}
}

func TestQueuedToFailed(t *testing.T) {
	st, ctx := setupTestStore(t)

	msg := core.NewMessage("OK", "body")
	msg.Tags = []string{"t_fail"}

	// Force deterministic failure:
	msg.Meta.Session = core.SessionRadioOnly
	msg.Meta.Transport.Allowed = []core.Mode{core.ModeTelnet}

	if err := st.SaveMessage(ctx, msg); err != nil {
		t.Fatalf("SaveMessage: %v", err)
	}

	n, err := st.QueueByTag(ctx, "t_fail")
	if err != nil {
		t.Fatalf("QueueByTag: %v", err)
	}
	if n != 1 {
		t.Fatalf("expected 1 queued, got %d", n)
	}

	res, err := ops.SendQueued(ctx, st, "t_fail", 10, sim.New())
	if err != nil {
		t.Fatalf("SendQueued: %v", err)
	}
	if res.Sent != 0 || res.Failed != 1 {
		t.Fatalf("expected sent=0 failed=1, got sent=%d failed=%d", res.Sent, res.Failed)
	}

	failed, err := st.ListByStatus(ctx, []core.MessageStatus{core.StatusFailed}, 10)
	if err != nil {
		t.Fatalf("ListByStatus(failed): %v", err)
	}
	if len(failed) != 1 {
		t.Fatalf("expected 1 failed message, got %d", len(failed))
	}
}
func TestOnlyQueuedMessagesAreSent(t *testing.T) {
	st, ctx := setupTestStore(t)

	queued := core.NewMessage("OK1", "body")
	queued.Tags = []string{"t_queue_me"}

	draft := core.NewMessage("OK2", "body")
	draft.Tags = []string{"t_do_not_queue"}

	if err := st.SaveMessage(ctx, queued); err != nil {
		t.Fatalf("SaveMessage(queued): %v", err)
	}
	if err := st.SaveMessage(ctx, draft); err != nil {
		t.Fatalf("SaveMessage(draft): %v", err)
	}

	n, err := st.QueueByTag(ctx, "t_queue_me")
	if err != nil {
		t.Fatalf("QueueByTag: %v", err)
	}
	if n != 1 {
		t.Fatalf("expected 1 queued, got %d", n)
	}

	res, err := ops.SendQueued(ctx, st, "", 10, sim.New())
	if err != nil {
		t.Fatalf("SendQueued: %v", err)
	}
	if res.Sent != 1 || res.Failed != 0 {
		t.Fatalf("expected sent=1 failed=0, got sent=%d failed=%d", res.Sent, res.Failed)
	}

	// Verify one is sent and the other remains draft
	sent, err := st.ListByStatus(ctx, []core.MessageStatus{core.StatusSent}, 10)
	if err != nil {
		t.Fatalf("ListByStatus(sent): %v", err)
	}
	if len(sent) != 1 {
		t.Fatalf("expected 1 sent, got %d", len(sent))
	}

	drafts, err := st.ListByStatus(ctx, []core.MessageStatus{core.StatusDraft}, 10)
	if err != nil {
		t.Fatalf("ListByStatus(draft): %v", err)
	}
	if len(drafts) != 1 {
		t.Fatalf("expected 1 draft remaining, got %d", len(drafts))
	}
}

func TestQueueByTagRequeuesFailed(t *testing.T) {
	st, ctx := setupTestStore(t)

	msg := core.NewMessage("OK", "body")
	msg.Tags = []string{"t_requeue"}

	if err := st.SaveMessage(ctx, msg); err != nil {
		t.Fatalf("SaveMessage: %v", err)
	}

	// Force it into failed state (this test is about re-queuing failed items)
	if err := st.SetStatusByID(ctx, msg.ID, core.StatusFailed, "forced failure"); err != nil {
		t.Fatalf("SetStatusByID(failed): %v", err)
	}

	// Sanity check: we have exactly one failed
	failed, err := st.ListByStatus(ctx, []core.MessageStatus{core.StatusFailed}, 10)
	if err != nil {
		t.Fatalf("ListByStatus(failed): %v", err)
	}
	if len(failed) != 1 {
		t.Fatalf("expected 1 failed, got %d", len(failed))
	}

	// Requeue should move failed -> queued (if QueueByTag supports draft/failed)
	n, err := st.QueueByTag(ctx, "t_requeue")
	if err != nil {
		t.Fatalf("QueueByTag(requeue): %v", err)
	}
	if n != 1 {
		t.Fatalf("expected 1 requeued, got %d", n)
	}

	queued, err := st.ListByStatus(ctx, []core.MessageStatus{core.StatusQueued}, 10)
	if err != nil {
		t.Fatalf("ListByStatus(queued): %v", err)
	}
	if len(queued) != 1 {
		t.Fatalf("expected 1 queued after requeue, got %d", len(queued))
	}
}
