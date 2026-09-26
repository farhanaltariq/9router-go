package chat

import (
	"testing"

	"9router/proxy/internal/db"
	"9router/proxy/internal/models"
)

func TestFilterConnectionsForModel_Conditions(t *testing.T) {
	conn1 := &models.ProviderConnection{ID: "c1", Data: `{"assignedModel":"gpt-4o"}`}
	conn2 := &models.ProviderConnection{ID: "c2", Data: `{"assignedModel":"o3-mini"}`}
	conn3 := &models.ProviderConnection{ID: "c3", Data: `{"providerSpecificData":{"assignedModel":"gpt-4o"}}`}
	conn4 := &models.ProviderConnection{ID: "c4", Data: `{"providerSpecificData":{"assignedModel":"claude-3-5-sonnet"}}`}
	conn5 := &models.ProviderConnection{ID: "c5", Data: `{"apiKey":"sk-none"}`}
	connNil := (*models.ProviderConnection)(nil)
	connMalformed := &models.ProviderConnection{ID: "c6", Data: `{invalid-json}`}

	all := []*models.ProviderConnection{conn1, conn2, conn3, conn4, conn5, connNil, connMalformed}

	// 1. settings == nil
	res := filterConnectionsForModel("openai", all, "gpt-4o", nil)
	if len(res) != len(all) {
		t.Fatalf("expected all connections when settings == nil, got %d", len(res))
	}

	// 2. settings.ProviderStrategies == nil
	res = filterConnectionsForModel("openai", all, "gpt-4o", &db.SettingsData{})
	if len(res) != len(all) {
		t.Fatalf("expected all connections when ProviderStrategies == nil, got %d", len(res))
	}

	// 3. model == ""
	settings := &db.SettingsData{
		ProviderStrategies: map[string]db.ProviderStrategy{
			"openai": {StrictModelAssignment: true},
		},
	}
	res = filterConnectionsForModel("openai", all, "", settings)
	if len(res) != len(all) {
		t.Fatalf("expected all connections when model is empty, got %d", len(res))
	}

	// 4. provider not in strategies
	res = filterConnectionsForModel("anthropic", all, "gpt-4o", settings)
	if len(res) != len(all) {
		t.Fatalf("expected all connections when provider not in strategies, got %d", len(res))
	}

	// 5. StrictModelAssignment == false
	settings.ProviderStrategies["openai"] = db.ProviderStrategy{StrictModelAssignment: false}
	res = filterConnectionsForModel("openai", all, "gpt-4o", settings)
	if len(res) != len(all) {
		t.Fatalf("expected all connections when StrictModelAssignment is false, got %d", len(res))
	}

	// 6. StrictModelAssignment == true, filter for gpt-4o
	settings.ProviderStrategies["openai"] = db.ProviderStrategy{StrictModelAssignment: true}
	res = filterConnectionsForModel("openai", all, "gpt-4o", settings)
	if len(res) != 2 {
		t.Fatalf("expected 2 connections for gpt-4o (conn1 and conn3), got %d", len(res))
	}
	if res[0].ID != "c1" || res[1].ID != "c3" {
		t.Errorf("unexpected connection IDs: %s, %s", res[0].ID, res[1].ID)
	}

	// 7. Filter for o3-mini
	res = filterConnectionsForModel("openai", all, "o3-mini", settings)
	if len(res) != 1 || res[0].ID != "c2" {
		t.Fatalf("expected 1 connection (c2) for o3-mini, got %v", res)
	}

	// 8. Filter for claude-3-5-sonnet
	res = filterConnectionsForModel("openai", all, "claude-3-5-sonnet", settings)
	if len(res) != 1 || res[0].ID != "c4" {
		t.Fatalf("expected 1 connection (c4) for claude-3-5-sonnet, got %v", res)
	}

	// 9. Filter for model with no matches
	res = filterConnectionsForModel("openai", all, "non-existent-model", settings)
	if len(res) != 0 {
		t.Fatalf("expected 0 connections for non-existent-model, got %d", len(res))
	}
}
