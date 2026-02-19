package store

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// ExternalRef links a RelayOps message (internal UUID) to a backend's identifier.
// Examples:
//
//	backend="pat",    external_id=<PAT MID>
//	backend="winlink", external_id=<Winlink Express msgid (filename stem)>
type ExternalRef struct {
	ID         string
	MessageID  string
	Backend    string
	ExternalID string
	Scope      string
	MetaJSON   string
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

func (s *Store) UpsertExternalRef(ctx context.Context, messageID, backend, externalID, scope, metaJSON string) error {
	if s == nil || s.db == nil {
		return fmt.Errorf("UpsertExternalRef: store is nil")
	}
	if messageID == "" || backend == "" || externalID == "" {
		return fmt.Errorf("UpsertExternalRef: messageID/backend/externalID required")
	}
	if metaJSON == "" {
		metaJSON = "{}"
	}

	now := time.Now().UTC().Format(time.RFC3339)

	// Ensure uniqueness by (backend, external_id, scope). If already exists, update message_id and metadata.
	_, err := s.db.ExecContext(ctx, `
	INSERT INTO message_external_refs(
		id, message_id, backend, external_id, scope, meta_json, created_at, updated_at
	) VALUES(?, ?, ?, ?, ?, ?, ?, ?)
	ON CONFLICT(backend, external_id, scope) DO UPDATE SET
		message_id = excluded.message_id,
		meta_json = excluded.meta_json,
		updated_at = excluded.updated_at
	`,
		uuid.NewString(), messageID, backend, externalID, scope, metaJSON, now, now,
	)
	if err != nil {
		return fmt.Errorf("UpsertExternalRef: %w", err)
	}
	return nil
}

// GetMessageIDByExternalRef returns the internal message_id for a given backend external reference.
func (s *Store) GetMessageIDByExternalRef(ctx context.Context, backend, externalID, scope string) (string, bool, error) {
	if s == nil || s.db == nil {
		return "", false, fmt.Errorf("GetMessageIDByExternalRef: store is nil")
	}
	row := s.db.QueryRowContext(ctx,
		`SELECT message_id FROM message_external_refs WHERE backend = ? AND external_id = ? AND scope = ?`,
		backend, externalID, scope,
	)
	var messageID string
	if err := row.Scan(&messageID); err != nil {
		if err == sql.ErrNoRows {
			return "", false, nil
		}
		return "", false, fmt.Errorf("GetMessageIDByExternalRef: %w", err)
	}
	return messageID, true, nil
}
