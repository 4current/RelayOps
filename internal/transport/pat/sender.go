package pat

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"

	"github.com/4current/relayops/internal/core"
)

type Sender struct {
	PatBinary       string // default "pat"
	DefaultFromCall string // e.g. "AE4OK"
	Service         string // e.g. "telnet"
}

func New(defaultFromCall string) *Sender {
	return &Sender{
		PatBinary:       "pat",
		DefaultFromCall: defaultFromCall,
		Service:         "telnet",
	}
}

func (s *Sender) SendOne(ctx context.Context, m *core.Message) (string, error) {
	cfg, _, err := LoadConfig()
	if err != nil {
		return "", err
	}

	outDir, err := OutboxDir(cfg.MyCall)
	if err != nil {
		return "", err
	}

	mid := NewMID(12)
	b2f, err := BuildB2F(cfg.MyCall, mid, m)
	if err != nil {
		return "", err
	}
	if _, err := WriteB2F(outDir, mid, b2f); err != nil {
		return "", err
	}
	// Run a connect to flush outbox.
	// Later we’ll optimize: send batches per session, not per message.
	args := []string{"connect", s.Service}
	cmd := exec.CommandContext(ctx, s.PatBinary, args...)

	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("pat connect failed: %w: %s", err, strings.TrimSpace(out.String()))
	}
	return mid, nil
}
