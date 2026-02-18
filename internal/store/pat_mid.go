package store

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/4current/relayops/internal/core"
)

func (s *Store) SetPatMIDByID(ctx context.Context, id, patMID string) error {
	// Deprecated: keep for compatibility with older code paths.
	// New code should call UpsertExternalRef(ctx, messageID, "pat", patMID, patService, metaJSON).
	_ = s.UpsertExternalRef(ctx, id, "pat", patMID, "", "{}")

	// Fetch current meta_json
	var metaJSON string
	if err := s.db.QueryRowContext(ctx,
		`SELECT meta_json FROM messages WHERE id = ?`, id,
	).Scan(&metaJSON); err != nil {
		return fmt.Errorf("SetPatMIDByID select meta_json: %w", err)
	}

	var meta core.MessageMeta
	if metaJSON != "" {
		if err := json.Unmarshal([]byte(metaJSON), &meta); err != nil {
			return fmt.Errorf("SetPatMIDByID unmarshal meta_json: %w", err)
		}
	}

	meta.Delivery.PatMID = patMID

	b, err := json.Marshal(meta)
	if err != nil {
		return fmt.Errorf("SetPatMIDByID marshal meta: %w", err)
	}

	_, err = s.db.ExecContext(ctx,
		`UPDATE messages SET meta_json = ? WHERE id = ?`,
		string(b), id,
	)
	if err != nil {
		return fmt.Errorf("SetPatMIDByID update meta_json: %w", err)
	}
	return nil
}
