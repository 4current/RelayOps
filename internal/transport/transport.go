package transport

import (
	"context"

	"github.com/4current/relayops/internal/core"
)

type Capability int

const (
	StoreAndForward Capability = iota
	Interactive
	Streaming
	Directory
)

type Transport interface {
	ID() string
	Capabilities() []Capability

	Available() bool
	Score(msg *core.Message) int

	Connect(ctx context.Context) error
	Send([]*core.Message) error
	Receive() ([]*core.Message, error)
	Disconnect() error
}
