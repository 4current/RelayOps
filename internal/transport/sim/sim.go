package sim

import (
	"context"
	"fmt"

	"github.com/4current/relayops/internal/core"
)

type Sender struct{}

func New() *Sender { return &Sender{} }

func (s *Sender) SendOne(ctx context.Context, m *core.Message) (string, error) {
	_ = ctx

	// Accept by default.
	if len(m.Meta.Transport.Allowed) == 0 || contains(m.Meta.Transport.Allowed, core.ModeAny) {
		return "SIM-OK", nil
	}

	// Deterministic failure used by tests:
	if m.Meta.Session == core.SessionRadioOnly &&
		len(m.Meta.Transport.Allowed) == 1 &&
		m.Meta.Transport.Allowed[0] == core.ModeTelnet {
		return "", fmt.Errorf("radio_only session cannot be telnet-only")
	}

	return "SIM-OK", nil
}

func contains(list []core.Mode, x core.Mode) bool {
	for _, m := range list {
		if m == x {
			return true
		}
	}
	return false
}
