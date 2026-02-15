package store_test

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/4current/relayops/internal/core"
	"github.com/4current/relayops/internal/store"
)

func setupStore(t *testing.T) (*store.Store, context.Context) {
	t.Helper()
	tmp := t.TempDir()
	_ = os.Setenv("HOME", tmp)
	_ = os.Setenv("USERPROFILE", tmp)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	t.Cleanup(cancel)

	st, err := store.Open(ctx) // covers migrations (v1/v2) path
	if err != nil {
		t.Fatalf("store.Open: %v", err)
	}
	t.Cleanup(func() { _ = st.Close() })
	return st, ctx
}

func TestSaveQueueListByStatus(t *testing.T) {
	st, ctx := setupStore(t)

	msg := core.NewMessage("Subj", "Body")
	msg.Tags = []string{"tagA"}
	msg.Meta.Transport.Allowed = []core.Mode{core.ModePacket}
	msg.Meta.Session = core.SessionWinlink

	if err := st.SaveMessage(ctx, msg); err != nil {
		t.Fatalf("SaveMessage: %v", err)
	}

	// Draft should show up
	drafts, err := st.ListByStatus(ctx, []core.MessageStatus{core.StatusDraft}, 10)
	if err != nil {
		t.Fatalf("ListByStatus(draft): %v", err)
	}
	if len(drafts) != 1 {
		t.Fatalf("expected 1 draft, got %d", len(drafts))
	}
	if drafts[0].Status != core.StatusDraft {
		t.Fatalf("expected status draft, got %s", drafts[0].Status)
	}

	// Queue by tag
	n, err := st.QueueByTag(ctx, "tagA")
	if err != nil {
		t.Fatalf("QueueByTag: %v", err)
	}
	if n != 1 {
		t.Fatalf("expected 1 queued, got %d", n)
	}

	queued, err := st.ListByStatus(ctx, []core.MessageStatus{core.StatusQueued}, 10)
	if err != nil {
		t.Fatalf("ListByStatus(queued): %v", err)
	}
	if len(queued) != 1 {
		t.Fatalf("expected 1 queued, got %d", len(queued))
	}
	if queued[0].Status != core.StatusQueued {
		t.Fatalf("expected status queued, got %s", queued[0].Status)
	}
}

func TestSetStatusByIDSentAndFailed(t *testing.T) {
	st, ctx := setupStore(t)

	msg := core.NewMessage("X", "Y")
	msg.Tags = []string{"t"}
	if err := st.SaveMessage(ctx, msg); err != nil {
		t.Fatalf("SaveMessage: %v", err)
	}

	// failed
	if err := st.SetStatusByID(ctx, msg.ID, core.StatusFailed, "boom"); err != nil {
		t.Fatalf("SetStatusByID(failed): %v", err)
	}
	failed, err := st.ListByStatus(ctx, []core.MessageStatus{core.StatusFailed}, 10)
	if err != nil {
		t.Fatalf("ListByStatus(failed): %v", err)
	}
	if len(failed) != 1 {
		t.Fatalf("expected 1 failed, got %d", len(failed))
	}

	// sent
	if err := st.SetStatusByID(ctx, msg.ID, core.StatusSent, ""); err != nil {
		t.Fatalf("SetStatusByID(sent): %v", err)
	}
	sent, err := st.ListByStatus(ctx, []core.MessageStatus{core.StatusSent}, 10)
	if err != nil {
		t.Fatalf("ListByStatus(sent): %v", err)
	}
	if len(sent) != 1 {
		t.Fatalf("expected 1 sent, got %d", len(sent))
	}
}

func TestListQueuedReturnsQueuedMessages(t *testing.T) {
	st, ctx := setupStore(t)

	m1 := core.NewMessage("A", "B")
	m1.Tags = []string{"qtag"}
	if err := st.SaveMessage(ctx, m1); err != nil {
		t.Fatalf("SaveMessage: %v", err)
	}

	_, err := st.QueueByTag(ctx, "qtag")
	if err != nil {
		t.Fatalf("QueueByTag: %v", err)
	}

	msgs, err := st.ListQueued(ctx, "qtag", 10)
	if err != nil {
		t.Fatalf("ListQueued: %v", err)
	}
	if len(msgs) != 1 {
		t.Fatalf("expected 1 queued message, got %d", len(msgs))
	}
	if msgs[0].Status != core.StatusQueued {
		t.Fatalf("expected queued status, got %s", msgs[0].Status)
	}
}
