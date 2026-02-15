package ops

import (
	"context"
	"fmt"

	"github.com/4current/relayops/internal/core"
	"github.com/4current/relayops/internal/store"
)

type Sender interface {
	SendOne(ctx context.Context, m *core.Message) error
}

type SendResult struct {
	Sent   int
	Failed int
}

func SendQueued(ctx context.Context, st *store.Store, tag string, limit int, sender Sender) (SendResult, error) {
	if st == nil {
		return SendResult{}, fmt.Errorf("store is nil")
	}
	if sender == nil {
		return SendResult{}, fmt.Errorf("sender is nil")
	}

	msgs, err := st.ListQueued(ctx, tag, limit)
	if err != nil {
		return SendResult{}, err
	}

	var res SendResult
	for _, m := range msgs {
		// basic sanity checks that should hold regardless of transport implementation
		if m.Meta.Session == core.SessionP2P && containsMode(m.Meta.Transport.Allowed, core.ModeTelnet) {
			_ = st.SetStatusByID(ctx, m.ID, core.StatusFailed, "session p2p incompatible with telnet allow-list")
			res.Failed++
			continue
		}

		if err := st.MarkSending(ctx, m.ID); err != nil {
			_ = st.SetStatusByID(ctx, m.ID, core.StatusFailed, err.Error())
			res.Failed++
			continue
		}

		if err := sender.SendOne(ctx, m); err != nil {
			_ = st.SetStatusByID(ctx, m.ID, core.StatusFailed, err.Error())
			res.Failed++
			continue
		}

		_ = st.SetStatusByID(ctx, m.ID, core.StatusSent, "")
		res.Sent++
	}

	return res, nil
}

func containsMode(list []core.Mode, x core.Mode) bool {
	for _, m := range list {
		if m == x {
			return true
		}
	}
	return false
}
