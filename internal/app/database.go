package app

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"

	"go.uber.org/fx"

	"9router/proxy/internal/config"
	"9router/proxy/internal/db"
	"9router/proxy/internal/dbtest"
	"9router/proxy/internal/log"
	"9router/proxy/internal/providers"
)

// DatabaseModule handles database initialization and provides *sql.DB and *db.Repo.
var DatabaseModule = fx.Module("database",
	fx.Provide(
		ProvideDatabase,
		ProvideRepo,
	),
)

// ProvideDatabase initializes the global SQLite database and registers an OnStop lifecycle hook to close it cleanly.
func ProvideDatabase(lc fx.Lifecycle, cfg *config.Config) (*sql.DB, error) {
	if err := db.InitGlobalDatabase(cfg.DatabasePath); err != nil {
		return nil, fmt.Errorf("database init: %w", err)
	}

	conn, err := db.GetConnection()
	if err != nil {
		return nil, fmt.Errorf("database connect: %w", err)
	}

	// Clean up orphaned WAL/SHM files from previous runs (fresh DB after rm data.sqlite).
	// These can cause schema mismatches or stale-state reads.
	for _, ext := range []string{"-wal", "-shm"} {
		_ = os.Remove(cfg.DatabasePath + ext)
	}

	// Auto-create dashboard tables if missing (fresh DB or after manual deletion).
	// Uses IF NOT EXISTS so it's safe to run repeatedly.
	for _, stmt := range dbtest.SchemaStatements() {
		if _, err := conn.Exec(stmt); err != nil {
			return nil, fmt.Errorf("create table: %w", err)
		}
	}
	// proxyPools table is not in dbtest.SchemaStatements but is required.
	if _, err := conn.Exec(`CREATE TABLE IF NOT EXISTS proxyPools (
		id TEXT PRIMARY KEY,
		isActive INTEGER DEFAULT 1,
		testStatus TEXT,
		data TEXT NOT NULL,
		createdAt TEXT NOT NULL,
		updatedAt TEXT NOT NULL
	)`); err != nil {
		return nil, fmt.Errorf("create proxyPools table: %w", err)
	}

	// Add missing columns to existing tables (schema drift from upstream).
	for _, alter := range []string{
		`ALTER TABLE providerConnections ADD COLUMN testStatus TEXT`,
		`ALTER TABLE providerConnections ADD COLUMN lastError TEXT`,
		`ALTER TABLE combos ADD COLUMN strategy TEXT`,
	} {
		if _, err := conn.Exec(alter); err != nil {
			// Column already exists or other error — ignore.
		}
	}

	// Wipe stale custom provider nodes whose prefix no longer maps to a known
	// provider. Keeps user-created OpenAI/Anthropic-compatible nodes intact.
	if err := cleanupStaleProviderNodes(conn); err != nil {
		log.Warn("startup", "provider node cleanup failed (non-fatal)", "err", err)
	}

	// Cross-process lease table for upstream coordination (Freebuff
	// sessions, future scopes). Idempotent: no-op when already present,
	// invisible to dashboards that do not know the table. Best-effort:
	// a shared test binary may hand us a connection bound to a removed
	// temp file (global singleton); leases then simply stay unavailable.
	if err := db.EnsureUpstreamLeases(conn); err != nil {
		_, statErr := conn.Exec("SELECT 1")
		if statErr == nil {
			return nil, fmt.Errorf("database leases: %w", err)
		}
	}

	lc.Append(fx.Hook{
		OnStop: func(ctx context.Context) error {
			return conn.Close()
		},
	})

	return conn, nil
}

// ProvideRepo provides *db.Repo using the database connection.
func ProvideRepo(conn *sql.DB) *db.Repo {
	return db.NewRepo(conn)
}

// knownProviderIDs lists the canonical provider IDs from the curated catalog.
// Custom provider nodes (openai-compatible, anthropic-compatible) are NOT in
// this list — they are kept regardless of their prefix.
var knownProviderIDs = map[string]bool{
	"antigravity":    true,
	"github":         true,
	"codex":          true,
	"codebuddy-intl": true,
	"codebuddy-cn":   true,
	"nvidia":         true,
	"ollama":         true,
	"cloudflare-ai":  true,
	"commandcode":    true,
	"opencode":       true,
}

// cleanupStaleProviderNodes removes providerNodes whose prefix does not match
// a known provider and is not an OpenAI/Anthropic-compatible custom node.
func cleanupStaleProviderNodes(conn *sql.DB) error {
	rows, err := conn.Query("SELECT id, data FROM providerNodes")
	if err != nil {
		return fmt.Errorf("query providerNodes: %w", err)
	}
	defer rows.Close()

	var toDelete []string
	for rows.Next() {
		var id, dataStr string
		if err := rows.Scan(&id, &dataStr); err != nil {
			return fmt.Errorf("scan providerNode: %w", err)
		}
		var data map[string]any
		if err := json.Unmarshal([]byte(dataStr), &data); err != nil {
			continue
		}
		prefix, _ := data["prefix"].(string)
		apiType, _ := data["apiType"].(string)
		// Keep OpenAI/Anthropic-compatible custom nodes.
		if apiType == "openai-compatible" || apiType == "anthropic-compatible" {
			continue
		}
		// Resolve alias to canonical provider ID.
		canonical := providers.ResolveAlias(prefix)
		if !knownProviderIDs[canonical] {
			toDelete = append(toDelete, id)
		}
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("iterate providerNodes: %w", err)
	}

	for _, id := range toDelete {
		if _, err := conn.Exec("DELETE FROM providerNodes WHERE id = ?", id); err != nil {
			log.Warn("cleanup", "failed to delete stale provider node", "id", id, "err", err)
		} else {
			log.Info("cleanup", "removed stale provider node", "id", id)
		}
	}
	return nil
}
