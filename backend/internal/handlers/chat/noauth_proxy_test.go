package chat

import (
	"testing"

	"9router/proxy/internal/db"
)

func TestResolveProviderProxyPoolID_CrossAlias(t *testing.T) {
	database, cleanup := setupChatTestDB(t)
	defer cleanup()

	if _, err := database.Exec(`CREATE TABLE IF NOT EXISTS settings (id INTEGER PRIMARY KEY, data TEXT);`); err != nil {
		t.Fatalf("create settings table: %v", err)
	}

	// User sets proxyPoolId under "antigravity", request asks for "ag"
	settingsJSON := `{
		"providerStrategies": {
			"antigravity": {
				"proxyPoolId": "pool-antigravity-123"
			}
		}
	}`
	if _, err := database.Exec(`INSERT INTO settings (id, data) VALUES (1, ?) ON CONFLICT(id) DO UPDATE SET data = excluded.data`, settingsJSON); err != nil {
		t.Fatalf("insert settings: %v", err)
	}

	repo := db.NewRepo(database)
	h := NewChatHandler(repo)

	// 1. Direct match on antigravity
	if got := h.ResolveProviderProxyPoolID("antigravity"); got != "pool-antigravity-123" {
		t.Errorf("expected pool-antigravity-123 for antigravity, got %q", got)
	}

	// 2. Alias match on ag (should find antigravity proxy pool)
	if got := h.ResolveProviderProxyPoolID("ag"); got != "pool-antigravity-123" {
		t.Errorf("expected ag to inherit pool-antigravity-123 from antigravity strategy, got %q", got)
	}
}
