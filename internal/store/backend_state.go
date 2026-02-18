package store

import (
	"context"
	"fmt"
	"time"
)

// UpsertBackendState stores per-backend folder/state information for a message.
// This is intentionally separate from core.Message.Status, which is RelayOps-local.
func (s *Store) UpsertBackendState(ctx context.Context, messageID, backend, folder, state, extraJSON string) error {
	if s == nil || s.db == nil {
		return fmt.Errorf("UpsertBackendState: store is nil")
	}
	if messageID == "" || backend == "" {
		return fmt.Errorf("UpsertBackendState: messageID/backend required")
	}
	if extraJSON == "" {
		extraJSON = "{}"
	}

	now := time.Now().UTC().Format(time.RFC3339)
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO message_backend_state(message_id, backend, folder, state, updated_at, extra_json)
		VALUES(?, ?, ?, ?, ?, ?)
		ON CONFLICT(message_id, backend) DO UPDATE SET
			folder = excluded.folder,
			state = excluded.state,
			updated_at = excluded.updated_at,
			extra_json = excluded.extra_json
	`, messageID, backend, folder, state, now, extraJSON)
	if err != nil {
		return fmt.Errorf("UpsertBackendState: %w", err)
	}
	return nil
}
