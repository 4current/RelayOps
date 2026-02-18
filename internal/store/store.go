package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	_ "modernc.org/sqlite"

	"github.com/4current/relayops/internal/core"
	"github.com/4current/relayops/internal/runtime"
	"github.com/google/uuid"
)

const (
	schemaV1 = 1
	schemaV2 = 2
	schemaV3 = 3
)

type Store struct {
	db *sql.DB
}

func Open(ctx context.Context) (*Store, error) {
	dbPath, err := runtime.DBPath()
	if err != nil {
		return nil, err
	}

	// modernc sqlite DSN is just a filepath for basic use.
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, err
	}

	// Basic sanity check
	if err := db.PingContext(ctx); err != nil {
		_ = db.Close()
		return nil, err
	}

	s := &Store{db: db}
	if err := s.migrate(ctx); err != nil {
		_ = db.Close()
		return nil, err
	}

	return s, nil
}

func (s *Store) Close() error {
	if s == nil || s.db == nil {
		return nil
	}
	return s.db.Close()
}

func (s *Store) migrate(ctx context.Context) error {
	stmts := []string{
		`PRAGMA journal_mode=WAL;`,
		`PRAGMA foreign_keys=ON;`,
		`CREATE TABLE IF NOT EXISTS schema_migrations (
			version INTEGER PRIMARY KEY,
			applied_at TEXT NOT NULL
		);`,
	}

	for _, q := range stmts {
		if _, err := s.db.ExecContext(ctx, q); err != nil {
			return fmt.Errorf("migrate: %w", err)
		}
	}

	// Very simple migration system: apply version 1 if not present
	applied1, err := s.hasMigration(ctx, schemaV1)
	if err != nil {
		return err
	}
	if !applied1 {
		// Schema v1
		schema := []string{
			`CREATE TABLE IF NOT EXISTS messages (
		id TEXT PRIMARY KEY,
		subject TEXT NOT NULL,
		body TEXT NOT NULL,
		created_at TEXT NOT NULL,
		updated_at TEXT NOT NULL,
		status TEXT NOT NULL,
		sent_at TEXT,
		last_error TEXT NOT NULL,
		from_callsign TEXT,
		from_email TEXT,
		to_json TEXT NOT NULL,
		tags_json TEXT NOT NULL,
		meta_json TEXT NOT NULL
		);`,
			`CREATE INDEX IF NOT EXISTS idx_messages_created_at ON messages(created_at);`,
		}

		tx, err := s.db.BeginTx(ctx, nil)
		if err != nil {
			return err
		}
		defer func() { _ = tx.Rollback() }()

		for _, q := range schema {
			if _, err := tx.ExecContext(ctx, q); err != nil {
				return fmt.Errorf("apply schema v1: %w", err)
			}
		}

		if _, err := tx.ExecContext(ctx,
			`INSERT INTO schema_migrations(version, applied_at) VALUES(?, ?)`,
			schemaV1, time.Now().UTC().Format(time.RFC3339),
		); err != nil {
			return fmt.Errorf("record migration: %w", err)
		}

		return tx.Commit()
	}

	// Check for v2, which adds a "priority" field to MessageMeta. This is just an example of how to handle schema changes that require data transformation.
	applied2, err := s.hasMigration(ctx, schemaV2)
	if err != nil {
		return err
	}
	if !applied2 {
		if err := s.applyV2(ctx); err != nil {
			return err
		}
	}

	applied3, err := s.hasMigration(ctx, schemaV3)
	if err != nil {
		return err
	}
	if !applied3 {
		if err := s.applyV3(ctx); err != nil {
			return err
		}
	}
	return nil
}

func (s *Store) applyV3(ctx context.Context) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	stmts := []string{
		`CREATE TABLE IF NOT EXISTS message_external_refs (
			id TEXT PRIMARY KEY,
			message_id TEXT NOT NULL,
			backend TEXT NOT NULL,
			external_id TEXT NOT NULL,
			scope TEXT NOT NULL DEFAULT '',
			meta_json TEXT NOT NULL DEFAULT '{}',
			created_at TEXT NOT NULL,
			updated_at TEXT NOT NULL,
			UNIQUE (backend, external_id, scope),
			FOREIGN KEY (message_id) REFERENCES messages(id) ON DELETE CASCADE
		);`,
		`CREATE INDEX IF NOT EXISTS idx_msg_external_refs_message_id ON message_external_refs(message_id);`,
		`CREATE INDEX IF NOT EXISTS idx_msg_external_refs_backend ON message_external_refs(backend);`,

		`CREATE TABLE IF NOT EXISTS message_backend_state (
			message_id TEXT NOT NULL,
			backend TEXT NOT NULL,
			folder TEXT NOT NULL DEFAULT '',
			state TEXT NOT NULL DEFAULT '',
			updated_at TEXT NOT NULL,
			extra_json TEXT NOT NULL DEFAULT '{}',
			PRIMARY KEY (message_id, backend),
			FOREIGN KEY (message_id) REFERENCES messages(id) ON DELETE CASCADE
		);`,
	}

	for _, q := range stmts {
		if _, err := tx.ExecContext(ctx, q); err != nil {
			return fmt.Errorf("apply schema v3: %w", err)
		}
	}

	// Backfill external refs from existing meta_json (PAT integration).
	rows, err := tx.QueryContext(ctx, `SELECT id, meta_json, from_callsign FROM messages`)
	if err != nil {
		return fmt.Errorf("apply v3: scan messages: %w", err)
	}
	defer rows.Close()

	now := time.Now().UTC().Format(time.RFC3339)
	for rows.Next() {
		var id, metaJSON, fromCallsign string
		if err := rows.Scan(&id, &metaJSON, &fromCallsign); err != nil {
			return fmt.Errorf("apply v3: row scan: %w", err)
		}

		var meta core.MessageMeta
		if metaJSON != "" {
			_ = json.Unmarshal([]byte(metaJSON), &meta) // best-effort
		}

		if meta.Delivery.PatMID == "" {
			continue
		}

		refID := uuid.NewString()
		scope := runtime.IdentityScope(fromCallsign)
		_, _ = tx.ExecContext(ctx,
			`INSERT OR IGNORE INTO message_external_refs(
				id, message_id, backend, external_id, scope, meta_json, created_at, updated_at
			) VALUES(?, ?, ?, ?, ?, ?, ?, ?)`,
			refID, id, "pat", meta.Delivery.PatMID, scope, "{}", now, now,
		)
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("apply v3: rows err: %w", err)
	}

	if _, err := tx.ExecContext(ctx,
		`INSERT INTO schema_migrations(version, applied_at) VALUES(?, ?)`,
		schemaV3, now,
	); err != nil {
		return fmt.Errorf("record migration v3: %w", err)
	}

	return tx.Commit()
}

func (s *Store) applyV2(ctx context.Context) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	stmts := []string{
		`ALTER TABLE messages ADD COLUMN status TEXT NOT NULL DEFAULT 'draft';`,
		`ALTER TABLE messages ADD COLUMN updated_at TEXT NOT NULL DEFAULT '';`,
		`ALTER TABLE messages ADD COLUMN sent_at TEXT;`,
		`ALTER TABLE messages ADD COLUMN last_error TEXT NOT NULL DEFAULT '';`,
		`CREATE INDEX IF NOT EXISTS idx_messages_status ON messages(status);`,
		`CREATE INDEX IF NOT EXISTS idx_messages_updated_at ON messages(updated_at);`,
	}

	for _, q := range stmts {
		// Some SQLite ALTER TABLE ADD COLUMN may fail if already applied; treat "duplicate column" as ok.
		if _, err := tx.ExecContext(ctx, q); err != nil {
			// modernc/sqlite error text varies; simplest approach: ignore if column exists
			msg := err.Error()
			if strings.Contains(msg, "duplicate column") || strings.Contains(msg, "already exists") {
				continue
			}
			return fmt.Errorf("apply schema v2: %w", err)
		}
	}

	if _, err := tx.ExecContext(ctx,
		`UPDATE messages SET updated_at = created_at WHERE updated_at = ''`,
	); err != nil {
		return fmt.Errorf("backfill updated_at: %w", err)
	}

	if _, err := tx.ExecContext(ctx,
		`INSERT INTO schema_migrations(version, applied_at) VALUES(?, ?)`,
		schemaV2, time.Now().UTC().Format(time.RFC3339),
	); err != nil {
		return fmt.Errorf("record migration v2: %w", err)
	}

	return tx.Commit()
}

func (s *Store) hasMigration(ctx context.Context, version int) (bool, error) {
	row := s.db.QueryRowContext(ctx, `SELECT COUNT(1) FROM schema_migrations WHERE version = ?`, version)
	var n int
	if err := row.Scan(&n); err != nil {
		return false, err
	}
	return n > 0, nil
}

func (s *Store) SaveMessage(ctx context.Context, msg *core.Message) error {
	if msg == nil {
		return fmt.Errorf("SaveMessage: msg is nil")
	}

	toJSON, err := json.Marshal(msg.To)
	if err != nil {
		return fmt.Errorf("SaveMessage: marshal To: %w", err)
	}
	tagsJSON, err := json.Marshal(msg.Tags)
	if err != nil {
		return fmt.Errorf("SaveMessage: marshal Tags: %w", err)
	}
	metaJSON, err := json.Marshal(msg.Meta)
	if err != nil {
		return fmt.Errorf("SaveMessage: marshal Meta: %w", err)
	}

	_, err = s.db.ExecContext(ctx, `
	INSERT INTO messages (
		id, subject, body, created_at,
		from_callsign, from_email,
		to_json, tags_json, meta_json,
		status, updated_at, sent_at, last_error
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
`,
		msg.ID, msg.Subject, msg.Body, msg.CreatedAt.UTC().Format(time.RFC3339),
		msg.From.Callsign, msg.From.Email,
		string(toJSON), string(tagsJSON), string(metaJSON),
		string(msg.Status),
		msg.UpdatedAt.UTC().Format(time.RFC3339),
		nil, // sent_at
		msg.LastError,
	)

	if err != nil {
		return fmt.Errorf("SaveMessage: %w", err)
	}
	return nil
}

type MessageSummary struct {
	ID        string
	Subject   string
	CreatedAt time.Time
	Tags      []string
	Meta      core.MessageMeta
	Status    core.MessageStatus
}

func (s *Store) ListMessages(ctx context.Context, limit int, includeDeleted bool) ([]MessageSummary, error) {
	if limit <= 0 {
		limit = 25
	}

	query := `SELECT id, subject, created_at, tags_json, meta_json, status FROM messages`
	if !includeDeleted {
		query += ` WHERE status != 'deleted'`
	}
	query += ` ORDER BY created_at DESC LIMIT ?`

	rows, err := s.db.QueryContext(ctx, query, limit)
	if err != nil {
		return nil, fmt.Errorf("ListMessages: %w", err)
	}
	defer rows.Close()

	var out []MessageSummary
	for rows.Next() {
		var id, subject, createdAtStr, tagsStr, metaStr, statusStr string
		if err := rows.Scan(&id, &subject, &createdAtStr, &tagsStr, &metaStr, &statusStr); err != nil {
			return nil, err
		}
		t, err := time.Parse(time.RFC3339, createdAtStr)
		if err != nil {
			t = time.Time{}
		}
		var tags []string
		_ = json.Unmarshal([]byte(tagsStr), &tags)

		var meta core.MessageMeta
		_ = json.Unmarshal([]byte(metaStr), &meta)

		out = append(out, MessageSummary{
			ID:        id,
			Subject:   subject,
			CreatedAt: t,
			Tags:      tags,
			Meta:      meta,
			Status:    core.MessageStatus(statusStr),
		})
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

func (s *Store) SetStatusByID(ctx context.Context, id string, status core.MessageStatus, lastErr string) error {
	now := time.Now().UTC().Format(time.RFC3339)

	var sentAt any = nil
	if status == core.StatusSent {
		sentAt = now
	}

	_, err := s.db.ExecContext(ctx, `
		UPDATE messages
		SET status = ?, updated_at = ?, sent_at = COALESCE(?, sent_at), last_error = ?
		WHERE id = ? AND status != 'deleted'
	`, string(status), now, sentAt, lastErr, id)

	if err != nil {
		return fmt.Errorf("SetStatusByID: %w", err)
	}
	return nil
}

func (s *Store) QueueByTag(ctx context.Context, tag string) (int64, error) {
	now := time.Now().UTC().Format(time.RFC3339)

	res, err := s.db.ExecContext(ctx, `
		UPDATE messages
		SET status = 'queued', updated_at = ?, last_error = ''
		WHERE status IN ('draft','failed')
		  AND (tags_json LIKE ?)
	`, now, "%"+tag+"%")
	if err != nil {
		return 0, fmt.Errorf("QueueByTag: %w", err)
	}
	return res.RowsAffected()
}

func (s *Store) ListByStatus(ctx context.Context, statuses []core.MessageStatus, limit int) ([]MessageSummary, error) {
	if limit <= 0 {
		limit = 25
	}
	if len(statuses) == 0 {
		statuses = []core.MessageStatus{core.StatusDraft, core.StatusQueued, core.StatusFailed}
	}

	// Build placeholders (?, ?, ?)
	ph := make([]string, 0, len(statuses))
	args := make([]any, 0, len(statuses)+1)
	for _, st := range statuses {
		ph = append(ph, "?")
		args = append(args, string(st))
	}
	args = append(args, limit)

	q := fmt.Sprintf(`
		SELECT id, subject, created_at, tags_json, meta_json, status
		FROM messages
		WHERE status IN (%s)
		ORDER BY updated_at DESC
		LIMIT ?
	`, strings.Join(ph, ","))

	rows, err := s.db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, fmt.Errorf("ListByStatus: %w", err)
	}
	defer rows.Close()

	var out []MessageSummary
	for rows.Next() {
		var id, subject, createdAtStr, tagsStr, metaStr, statusStr string
		if err := rows.Scan(&id, &subject, &createdAtStr, &tagsStr, &metaStr, &statusStr); err != nil {
			return nil, err
		}
		t, _ := time.Parse(time.RFC3339, createdAtStr)

		var tags []string
		_ = json.Unmarshal([]byte(tagsStr), &tags)

		var meta core.MessageMeta
		_ = json.Unmarshal([]byte(metaStr), &meta)

		out = append(out, MessageSummary{
			ID:        id,
			Subject:   subject,
			CreatedAt: t,
			Tags:      tags,
			Meta:      meta,
			Status:    core.MessageStatus(statusStr),
		})
	}
	return out, rows.Err()
}

func (s *Store) ListQueued(ctx context.Context, tag string, limit int) ([]*core.Message, error) {
	if limit <= 0 {
		limit = 25
	}

	args := []any{}
	where := "WHERE status = 'queued'"
	if strings.TrimSpace(tag) != "" {
		where += " AND tags_json LIKE ?"
		args = append(args, "%"+tag+"%")
	}
	args = append(args, limit)

	q := fmt.Sprintf(`
		SELECT id, subject, body, created_at, from_callsign, from_email,
		       to_json, tags_json, meta_json,
		       status, updated_at, sent_at, last_error
		FROM messages
		%s
		ORDER BY updated_at ASC
		LIMIT ?
	`, where)

	rows, err := s.db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, fmt.Errorf("ListQueued: %w", err)
	}
	defer rows.Close()

	var out []*core.Message
	for rows.Next() {
		var (
			id, subject, body, createdAtStr string
			fromCall, fromEmail             sql.NullString
			toStr, tagsStr, metaStr         string
			statusStr, updatedAtStr         string
			sentAtStr                       sql.NullString
			lastErr                         string
		)

		if err := rows.Scan(&id, &subject, &body, &createdAtStr, &fromCall, &fromEmail,
			&toStr, &tagsStr, &metaStr,
			&statusStr, &updatedAtStr, &sentAtStr, &lastErr,
		); err != nil {
			return nil, err
		}

		createdAt, _ := time.Parse(time.RFC3339, createdAtStr)
		updatedAt, _ := time.Parse(time.RFC3339, updatedAtStr)

		var to []core.Address
		_ = json.Unmarshal([]byte(toStr), &to)

		var tags []string
		_ = json.Unmarshal([]byte(tagsStr), &tags)

		var meta core.MessageMeta
		_ = json.Unmarshal([]byte(metaStr), &meta)

		var sentAtPtr *time.Time
		if sentAtStr.Valid {
			t, err := time.Parse(time.RFC3339, sentAtStr.String)
			if err == nil {
				sentAtPtr = &t
			}
		}

		out = append(out, &core.Message{
			ID:        id,
			Subject:   subject,
			Body:      body,
			CreatedAt: createdAt,
			UpdatedAt: updatedAt,
			From: core.Address{
				Callsign: fromCall.String,
				Email:    fromEmail.String,
			},
			To:        to,
			Tags:      tags,
			Meta:      meta,
			Status:    core.MessageStatus(statusStr),
			SentAt:    sentAtPtr,
			LastError: lastErr,
		})
	}
	return out, rows.Err()
}

func (s *Store) MarkSending(ctx context.Context, id string) error {
	return s.SetStatusByID(ctx, id, core.StatusSending, "")
}

func (s *Store) DeleteByID(ctx context.Context, id string) (int64, error) {
	res, err := s.db.ExecContext(ctx,
		`UPDATE messages SET status = ? WHERE id = ? AND status != ?`,
		core.StatusDeleted, id, core.StatusDeleted,
	)
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}
