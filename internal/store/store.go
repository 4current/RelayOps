package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	_ "modernc.org/sqlite"

	"github.com/4current/relayops/internal/core"
	"github.com/4current/relayops/internal/runtime"
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
	const v1 = 1
	applied, err := s.hasMigration(ctx, v1)
	if err != nil {
		return err
	}
	if applied {
		return nil
	}

	// Schema v1
	schema := []string{
		`CREATE TABLE IF NOT EXISTS messages (
			id TEXT PRIMARY KEY,
			subject TEXT NOT NULL,
			body TEXT NOT NULL,
			created_at TEXT NOT NULL,
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
		v1, time.Now().UTC().Format(time.RFC3339),
	); err != nil {
		return fmt.Errorf("record migration: %w", err)
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
			to_json, tags_json, meta_json
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, msg.ID, msg.Subject, msg.Body, msg.CreatedAt.UTC().Format(time.RFC3339),
		msg.From.Callsign, msg.From.Email,
		string(toJSON), string(tagsJSON), string(metaJSON),
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
}

func (s *Store) ListMessages(ctx context.Context, limit int) ([]MessageSummary, error) {
	if limit <= 0 {
		limit = 25
	}

	rows, err := s.db.QueryContext(ctx, `
		SELECT id, subject, created_at, tags_json
		FROM messages
		ORDER BY created_at DESC
		LIMIT ?
	`, limit)
	if err != nil {
		return nil, fmt.Errorf("ListMessages: %w", err)
	}
	defer rows.Close()

	var out []MessageSummary
	for rows.Next() {
		var id, subject, createdAtStr, tagsStr string
		if err := rows.Scan(&id, &subject, &createdAtStr, &tagsStr); err != nil {
			return nil, err
		}
		t, err := time.Parse(time.RFC3339, createdAtStr)
		if err != nil {
			t = time.Time{}
		}
		var tags []string
		_ = json.Unmarshal([]byte(tagsStr), &tags)

		out = append(out, MessageSummary{
			ID:        id,
			Subject:   subject,
			CreatedAt: t,
			Tags:      tags,
		})
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}
