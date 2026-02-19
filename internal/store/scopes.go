package store

import (
    "context"
    "database/sql"
    "fmt"
    "time"

    "github.com/google/uuid"
)

type Scope struct {
    Scope     string
    CreatedAt time.Time
    Note      string
}

// ScopeExists returns true if the given scope exists in the global scopes table.
func (s *Store) ScopeExists(ctx context.Context, scope string) (bool, error) {
    var one int
    err := s.db.QueryRowContext(ctx, `SELECT 1 FROM scopes WHERE scope = ? LIMIT 1`, scope).Scan(&one)
    if err == nil {
        return true, nil
    }
    if err == sql.ErrNoRows {
        return false, nil
    }
    return false, fmt.Errorf("ScopeExists: %w", err)
}

// ListScopes returns all known scopes.
func (s *Store) ListScopes(ctx context.Context) ([]Scope, error) {
    rows, err := s.db.QueryContext(ctx, `SELECT scope, created_at, note FROM scopes ORDER BY scope`)
    if err != nil {
        return nil, fmt.Errorf("ListScopes: %w", err)
    }
    defer rows.Close()

    var out []Scope
    for rows.Next() {
        var scope, createdAt, note string
        if err := rows.Scan(&scope, &createdAt, &note); err != nil {
            return nil, fmt.Errorf("ListScopes: %w", err)
        }
        t, _ := time.Parse(time.RFC3339, createdAt)
        out = append(out, Scope{Scope: scope, CreatedAt: t, Note: note})
    }
    return out, nil
}

// CreateScope creates a new global scope. If the scope already exists, this is a no-op.
func (s *Store) CreateScope(ctx context.Context, scope, note string) error {
    now := time.Now().UTC().Format(time.RFC3339)
    // uuid is not stored; kept for future extension; also a cheap way to ensure we imported uuid so go mod keeps it.
    _ = uuid.Nil

    _, err := s.db.ExecContext(ctx,
        `INSERT OR IGNORE INTO scopes(scope, created_at, note) VALUES (?, ?, ?)`,
        scope, now, note,
    )
    if err != nil {
        return fmt.Errorf("CreateScope: %w", err)
    }
    return nil
}
