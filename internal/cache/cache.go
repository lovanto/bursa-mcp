// Package cache is a small SQLite-backed key/value store with per-entry TTL.
// It exists so the MCP server can honour IDX's rate limits: financial reports
// are immutable and cached indefinitely, trading data for ~a day, etc. Uses the
// pure-Go modernc.org/sqlite driver to keep idx-mcp a single static binary
// (no cgo).
package cache

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	_ "modernc.org/sqlite"
)

// Cache stores byte payloads keyed by an arbitrary string, each with an
// absolute expiry timestamp.
type Cache struct {
	db *sql.DB
}

// Open opens (or creates) the SQLite database at path and ensures the schema
// exists. Use ":memory:" for tests.
func Open(path string) (*Cache, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}
	// modernc's driver is safe for concurrent use, but a single writer avoids
	// SQLITE_BUSY under the MCP server's serialized access pattern.
	db.SetMaxOpenConns(1)

	c := &Cache{db: db}
	if err := c.init(context.Background()); err != nil {
		db.Close()
		return nil, err
	}
	return c, nil
}

func (c *Cache) init(ctx context.Context) error {
	const schema = `
CREATE TABLE IF NOT EXISTS cache_entries (
	key        TEXT PRIMARY KEY,
	value      BLOB NOT NULL,
	expires_at INTEGER NOT NULL
);`
	if _, err := c.db.ExecContext(ctx, schema); err != nil {
		return fmt.Errorf("init schema: %w", err)
	}
	return nil
}

// Get returns the cached value for key if present and not expired. The second
// result is false on a miss or an expired entry.
func (c *Cache) Get(ctx context.Context, key string) ([]byte, bool, error) {
	var (
		value     []byte
		expiresAt int64
	)
	err := c.db.QueryRowContext(ctx,
		`SELECT value, expires_at FROM cache_entries WHERE key = ?`, key).
		Scan(&value, &expiresAt)
	if err == sql.ErrNoRows {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, fmt.Errorf("cache get: %w", err)
	}
	if time.Now().Unix() >= expiresAt {
		// Lazily evict; ignore delete errors since the read already missed.
		_, _ = c.db.ExecContext(ctx, `DELETE FROM cache_entries WHERE key = ?`, key)
		return nil, false, nil
	}
	return value, true, nil
}

// Set stores value under key with the given TTL. A ttl <= 0 stores the entry
// effectively forever (used for immutable financial reports).
func (c *Cache) Set(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	var expiresAt int64
	if ttl <= 0 {
		expiresAt = time.Date(9999, 1, 1, 0, 0, 0, 0, time.UTC).Unix()
	} else {
		expiresAt = time.Now().Add(ttl).Unix()
	}
	_, err := c.db.ExecContext(ctx,
		`INSERT INTO cache_entries (key, value, expires_at) VALUES (?, ?, ?)
		 ON CONFLICT(key) DO UPDATE SET value = excluded.value, expires_at = excluded.expires_at`,
		key, value, expiresAt)
	if err != nil {
		return fmt.Errorf("cache set: %w", err)
	}
	return nil
}

// Close releases the database handle.
func (c *Cache) Close() error { return c.db.Close() }
