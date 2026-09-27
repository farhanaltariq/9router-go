package dashboard

import (
	"context"
	json "encoding/json/v2"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"9router/proxy/internal/db"
	internalproxy "9router/proxy/internal/proxy"
	"9router/proxy/internal/proxy/oauth"
)

// setupProbeTestDB adds the tables the connection probe needs on top of the
// shared dashboard schema.
func setupProbeTestDB(t *testing.T) (*db.Repo, func()) {
	t.Helper()
	repo, cleanup := setupTestDB(t)
	stmts := []string{
		`CREATE TABLE IF NOT EXISTS proxyPools (
			id TEXT PRIMARY KEY,
			isActive INTEGER DEFAULT 1,
			testStatus TEXT,
			data TEXT NOT NULL,
			createdAt TEXT NOT NULL,
			updatedAt TEXT NOT NULL
		);`,
		`CREATE TABLE IF NOT EXISTS providerNodes (
			id TEXT PRIMARY KEY,
			type TEXT,
			name TEXT,
			data TEXT NOT NULL,
			createdAt TEXT NOT NULL,
			updatedAt TEXT NOT NULL
		);`,
	}
	for _, stmt := range stmts {
		if _, err := repo.RawDB().Exec(stmt); err != nil {
			cleanup()
			t.Fatalf("failed to create table: %v", err)
		}
	}
	return repo, cleanup
}

// stubProbeDo replaces the probe HTTP call for the duration of one test.
func stubProbeDo(t *testing.T, fn func(method, rawURL string) (int, []byte)) {
	t.Helper()
	original := connectionProbeDo
	connectionProbeDo = func(_ context.Context, _ *http.Client, method, rawURL string, _ map[string]string, _ []byte) (int, []byte, error) {
		status, body := fn(method, rawURL)
		return status, body, nil
	}
	t.Cleanup(func() { connectionProbeDo = original })
}

func postTestConnection(t *testing.T, repo *db.Repo, id string) (int, map[string]any) {
	t.Helper()
	router := setupTestRouter(repo)
	req := httptest.NewRequest(http.MethodPost, "/api/providers/"+id+"/test", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	var body map[string]any
	_ = json.Unmarshal(rec.Body.Bytes(), &body)
	return rec.Code, body
}

func readConnectionData(t *testing.T, repo *db.Repo, id string) map[string]any {
	t.Helper()
	conn, err := repo.GetProviderConnectionByID(id)
	if err != nil || conn == nil {
		t.Fatalf("failed to read connection %s: %v", id, err)
	}
	var data map[string]any
	if conn.Data != "" {
		_ = json.Unmarshal([]byte(conn.Data), &data)
	}
	return data
}

func TestTestConnection_NotFound(t *testing.T) {
	repo, cleanup := setupProbeTestDB(t)
	defer cleanup()

	status, body := postTestConnection(t, repo, "missing-connection")
	if status != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", status)
	}
	if body["valid"] != false || body["error"] != "Connection not found" {
		t.Errorf("unexpected body: %v", body)
	}
}

func TestTestConnection_OpenAICompatibleNode(t *testing.T) {
	repo, cleanup := setupProbeTestDB(t)
	defer cleanup()

	const id = "openai-compatible-chat-probe1"
	data := `{"apiKey":"sk-live","providerSpecificData":{"baseUrl":"https://node.example.com/v1"}}`
	if err := repo.CreateProviderConnectionFull(id, id, "apikey", "Node", nil, data); err != nil {
		t.Fatalf("create connection: %v", err)
	}

	var gotURL string
	stubProbeDo(t, func(method, rawURL string) (int, []byte) {
		gotURL = rawURL
		return http.StatusOK, []byte(`{"data":[]}`)
	})

	status, body := postTestConnection(t, repo, id)
	if status != http.StatusOK || body["valid"] != true {
		t.Fatalf("status/body = %d %v", status, body)
	}
	if gotURL != "https://node.example.com/v1/models" {
		t.Errorf("probed %q, want the node /models endpoint", gotURL)
	}
	updated := readConnectionData(t, repo, id)
	if updated["testStatus"] != "active" {
		t.Errorf("testStatus = %v, want active", updated["testStatus"])
	}
}

func TestTestConnection_OpenAICompatibleNodeRejectsBadKey(t *testing.T) {
	repo, cleanup := setupProbeTestDB(t)
	defer cleanup()

	const id = "openai-compatible-chat-probe2"
	data := `{"apiKey":"sk-bad","providerSpecificData":{"baseUrl":"https://node.example.com/v1/"}}`
	if err := repo.CreateProviderConnectionFull(id, id, "apikey", "Node", nil, data); err != nil {
		t.Fatalf("create connection: %v", err)
	}
	stubProbeDo(t, func(string, string) (int, []byte) { return http.StatusUnauthorized, nil })

	status, body := postTestConnection(t, repo, id)
	if status != http.StatusOK || body["valid"] != false {
		t.Fatalf("status/body = %d %v", status, body)
	}
	if body["error"] != "Invalid API key or base URL" {
		t.Errorf("error = %v", body["error"])
	}
	updated := readConnectionData(t, repo, id)
	if updated["testStatus"] != "error" || updated["lastError"] != "Invalid API key or base URL" {
		t.Errorf("unexpected write-back: %v", updated)
	}
	if updated["lastErrorAt"] == nil {
		t.Error("expected lastErrorAt to be set on failure")
	}
}

func TestTestConnection_AnthropicCompatibleNode(t *testing.T) {
	repo, cleanup := setupProbeTestDB(t)
	defer cleanup()

	const id = "anthropic-compatible-chat-probe3"
	data := `{"apiKey":"sk-ant","providerSpecificData":{"baseUrl":"https://node.example.com/v1/messages"}}`
	if err := repo.CreateProviderConnectionFull(id, id, "apikey", "Node", nil, data); err != nil {
		t.Fatalf("create connection: %v", err)
	}

	var gotURL string
	var gotBody string
	original := connectionProbeDo
	connectionProbeDo = func(_ context.Context, _ *http.Client, method, rawURL string, _ map[string]string, body []byte) (int, []byte, error) {
		gotURL = rawURL
		gotBody = string(body)
		return http.StatusBadRequest, nil, nil
	}
	t.Cleanup(func() { connectionProbeDo = original })

	_, body := postTestConnection(t, repo, id)
	// 400 still proves the key was accepted (upstream parity).
	if body["valid"] != true {
		t.Fatalf("expected valid for 400, got %v", body)
	}
	if gotURL != "https://node.example.com/v1/messages" {
		t.Errorf("probed %q, want /v1/messages with the /messages suffix collapsed", gotURL)
	}
	if !strings.Contains(gotBody, connectionAnthropicProbeModel) {
		t.Errorf("probe body missing default model: %s", gotBody)
	}

	// And a rejected key must fail.
	connectionProbeDo = func(_ context.Context, _ *http.Client, _, _ string, _ map[string]string, _ []byte) (int, []byte, error) {
		return http.StatusUnauthorized, nil, nil
	}
	_, body = postTestConnection(t, repo, id)
	if body["valid"] != false || body["error"] != "Invalid API key or base URL" {
		t.Errorf("expected invalid key result, got %v", body)
	}
}

func TestTestConnection_CompatibleNodeFromNodeRow(t *testing.T) {
	repo, cleanup := setupProbeTestDB(t)
	defer cleanup()

	const nodeID = "openai-compatible-chat-node9"
	if _, err := repo.CreateProviderNode(nodeID, "openai-compatible", "Node 9", `{"prefix":"n9","baseUrl":"https://from-node.example.com/v1"}`); err != nil {
		t.Fatalf("create node: %v", err)
	}
	if err := repo.CreateProviderConnectionFull(nodeID, nodeID, "apikey", "Node Key", nil, `{"apiKey":"sk-live"}`); err != nil {
		t.Fatalf("create connection: %v", err)
	}

	var gotURL string
	stubProbeDo(t, func(_, rawURL string) (int, []byte) {
		gotURL = rawURL
		return http.StatusOK, nil
	})

	if _, body := postTestConnection(t, repo, nodeID); body["valid"] != true {
		t.Fatalf("expected valid, got %v", body)
	}
	if gotURL != "https://from-node.example.com/v1/models" {
		t.Errorf("probed %q, want the node's base URL", gotURL)
	}
}

func TestTestConnection_APIKeyProviderReusesKeyProbe(t *testing.T) {
	repo, cleanup := setupProbeTestDB(t)
	defer cleanup()

	if err := repo.CreateProviderConnectionFull("conn-openai", "openai", "apikey", "OpenAI", nil, `{"apiKey":"sk-test"}`); err != nil {
		t.Fatalf("create connection: %v", err)
	}

	original := validateProbeDo
	defer func() { validateProbeDo = original }()

	validateProbeDo = func(context.Context, string, string, map[string]string, []byte) (int, []byte, error) {
		return http.StatusUnauthorized, nil, nil
	}
	if _, body := postTestConnection(t, repo, "conn-openai"); body["valid"] != false || body["error"] != "Invalid API key" {
		t.Fatalf("expected invalid key, got %v", body)
	}

	validateProbeDo = func(context.Context, string, string, map[string]string, []byte) (int, []byte, error) {
		return http.StatusOK, nil, nil
	}
	_, body := postTestConnection(t, repo, "conn-openai")
	if body["valid"] != true {
		t.Fatalf("expected valid, got %v", body)
	}
	if data := readConnectionData(t, repo, "conn-openai"); data["lastError"] != nil {
		t.Errorf("expected cleared lastError after success, got %v", data["lastError"])
	}
}

func TestTestConnection_UnknownProviderIsUnsupported(t *testing.T) {
	repo, cleanup := setupProbeTestDB(t)
	defer cleanup()

	if err := repo.CreateProviderConnectionFull("conn-unknown", "totally-unknown", "apikey", "X", nil, `{"apiKey":"k"}`); err != nil {
		t.Fatalf("create connection: %v", err)
	}
	_, body := postTestConnection(t, repo, "conn-unknown")
	if body["valid"] != false || body["error"] != "Provider test not supported" {
		t.Errorf("unexpected body: %v", body)
	}
}

func TestTestConnection_OAuthTokenExists(t *testing.T) {
	repo, cleanup := setupProbeTestDB(t)
	defer cleanup()

	if err := repo.CreateProviderConnectionFull("conn-cb", "codebuddy-cn", "oauth", "CodeBuddy", nil, `{"accessToken":"cb-token"}`); err != nil {
		t.Fatalf("create connection: %v", err)
	}
	_, body := postTestConnection(t, repo, "conn-cb")
	if body["valid"] != true {
		t.Fatalf("expected valid, got %v", body)
	}
}

func TestTestConnection_OAuthExpiredWithoutRefreshToken(t *testing.T) {
	repo, cleanup := setupProbeTestDB(t)
	defer cleanup()

	expired := time.Now().Add(-2 * time.Hour).UTC().Format(time.RFC3339)
	payload := `{"accessToken":"old","expiresAt":"` + expired + `"}`
	if err := repo.CreateProviderConnectionFull("conn-claude", "claude", "oauth", "Claude", nil, payload); err != nil {
		t.Fatalf("create connection: %v", err)
	}

	_, body := postTestConnection(t, repo, "conn-claude")
	if body["valid"] != false || body["error"] != "Token expired" {
		t.Fatalf("unexpected body: %v", body)
	}
}

func TestTestConnection_OAuthRefreshesExpiredToken(t *testing.T) {
	repo, cleanup := setupProbeTestDB(t)
	defer cleanup()

	// Registry refreshers are process-wide; this stub proves the refresh path
	// without touching the network and is scoped to this test's provider id.
	const provider = "probe-refresh-provider"
	original := oauth.Get(provider)
	oauth.Register(provider, func(_ context.Context, _ *oauth.Params) (*oauth.TokenResult, error) {
		return &oauth.TokenResult{AccessToken: "fresh-token", RefreshToken: "fresh-refresh", ExpiresIn: 3600}, nil
	})
	t.Cleanup(func() {
		if original != nil {
			oauth.Register(provider, original)
		}
	})

	expired := time.Now().Add(-2 * time.Hour).UTC().Format(time.RFC3339)
	payload := `{"accessToken":"old","refreshToken":"rt","expiresAt":"` + expired + `"}`
	if err := repo.CreateProviderConnectionFull("conn-refresh", provider, "oauth", "Refresh", nil, payload); err != nil {
		t.Fatalf("create connection: %v", err)
	}

	// checkExpiry providers validate purely through the refresh result.
	oauthProbeConfigs[provider] = oauthProbeConfig{checkExpiry: true, refreshable: true}
	t.Cleanup(func() { delete(oauthProbeConfigs, provider) })

	_, body := postTestConnection(t, repo, "conn-refresh")
	if body["valid"] != true || body["refreshed"] != true {
		t.Fatalf("unexpected body: %v", body)
	}
	data := readConnectionData(t, repo, "conn-refresh")
	if data["accessToken"] != "fresh-token" || data["refreshToken"] != "fresh-refresh" {
		t.Errorf("refreshed tokens not persisted: %v", data)
	}
	if data["expiresAt"] == expired {
		t.Error("expected expiresAt to be updated after refresh")
	}
}

func TestTestConnection_OAuthRefreshRetryOn401(t *testing.T) {
	repo, cleanup := setupProbeTestDB(t)
	defer cleanup()

	const provider = "codex"
	if err := repo.CreateProviderConnectionFull("conn-codex-401", provider, "oauth", "Codex", nil,
		`{"accessToken":"dead","refreshToken":"rt","expiresAt":"2099-01-01T00:00:00Z"}`); err != nil {
		t.Fatalf("create connection: %v", err)
	}

	original := oauth.Get(provider)
	oauth.Register(provider, func(_ context.Context, _ *oauth.Params) (*oauth.TokenResult, error) {
		return &oauth.TokenResult{AccessToken: "live-token"}, nil
	})
	t.Cleanup(func() {
		if original != nil {
			oauth.Register(provider, original)
		}
	})

	calls := 0
	originalProbe := connectionProbeDo
	connectionProbeDo = func(_ context.Context, _ *http.Client, _, rawURL string, headers map[string]string, _ []byte) (int, []byte, error) {
		calls++
		auth := headers["Authorization"]
		if strings.HasPrefix(auth, "Bearer live-token") {
			return http.StatusOK, nil, nil
		}
		return http.StatusUnauthorized, nil, nil
	}
	t.Cleanup(func() { connectionProbeDo = originalProbe })

	_, body := postTestConnection(t, repo, "conn-codex-401")
	if body["valid"] != true || body["refreshed"] != true {
		t.Fatalf("unexpected body: %v", body)
	}
	if calls < 2 {
		t.Errorf("expected a retry after refresh, calls = %d", calls)
	}
	if data := readConnectionData(t, repo, "conn-codex-401"); data["accessToken"] != "live-token" {
		t.Errorf("refreshed token not persisted: %v", data)
	}
}

func TestTestConnection_ProxyPrecheckFailure(t *testing.T) {
	repo, cleanup := setupProbeTestDB(t)
	defer cleanup()

	// Unreachable local proxy: the pre-check must fail before any provider call.
	payload := `{"apiKey":"sk-live","connectionProxyEnabled":true,"connectionProxyUrl":"http://127.0.0.1:1","providerSpecificData":{"baseUrl":"https://node.example.com/v1"}}`
	if err := repo.CreateProviderConnectionFull("conn-proxy", "openai-compatible-chat-proxy", "apikey", "Node", nil, payload); err != nil {
		t.Fatalf("create connection: %v", err)
	}

	probed := false
	stubProbeDo(t, func(string, string) (int, []byte) {
		probed = true
		return http.StatusOK, nil
	})

	_, body := postTestConnection(t, repo, "conn-proxy")
	if body["valid"] != false {
		t.Fatalf("expected proxy failure, got %v", body)
	}
	if probed {
		t.Error("provider must not be probed when the proxy pre-check fails")
	}
	if msg, _ := body["error"].(string); !strings.HasPrefix(msg, "Proxy test failed") {
		t.Errorf("error = %v", body["error"])
	}
	if data := readConnectionData(t, repo, "conn-proxy"); data["testStatus"] != "error" {
		t.Errorf("testStatus = %v, want error", data["testStatus"])
	}
}

func TestClassifyOAuthProbe(t *testing.T) {
	tests := []struct {
		name       string
		status     int
		cfg        oauthProbeConfig
		wantValid  bool
		wantStatus int
		wantMsg    string
	}{
		{name: "ok", status: 200, wantValid: true, wantStatus: 200},
		{name: "unauthorized", status: 401, wantStatus: 401, wantMsg: "Token invalid or revoked"},
		{name: "forbidden", status: 403, wantStatus: 403, wantMsg: "Access denied"},
		{name: "other", status: 500, wantStatus: 500, wantMsg: "API returned 500"},
		{name: "accepted 400 counts as valid", status: 400, cfg: oauthProbeConfig{acceptStatuses: []int{400}}, wantValid: true, wantStatus: 400},
		{
			name: "soft fail keeps valid with warning", status: 402,
			cfg:       oauthProbeConfig{acceptStatuses: []int{402}, softFailMessage: map[int]string{402: "credits exhausted"}},
			wantValid: true, wantStatus: 402, wantMsg: "credits exhausted",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := classifyOAuthProbe(tt.status, tt.cfg)
			if got.valid != tt.wantValid || got.status != tt.wantStatus || got.message != tt.wantMsg {
				t.Errorf("classifyOAuthProbe(%d) = %+v, want valid=%v status=%d msg=%q", tt.status, got, tt.wantValid, tt.wantStatus, tt.wantMsg)
			}
		})
	}
}

func TestConnectionTokenExpired(t *testing.T) {
	now := time.Now().UTC()
	tests := []struct {
		name      string
		expiresAt string
		want      bool
	}{
		{name: "missing expiresAt is not expired", want: false},
		{name: "unparsable expiresAt is not expired", expiresAt: "not-a-date", want: false},
		{name: "past", expiresAt: now.Add(-time.Hour).Format(time.RFC3339), want: true},
		{name: "inside the refresh lead", expiresAt: now.Add(time.Minute).Format(time.RFC3339), want: true},
		{name: "far future", expiresAt: now.Add(time.Hour).Format(time.RFC3339), want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := connectionTokenExpired(connectionProbeData{ExpiresAt: tt.expiresAt}); got != tt.want {
				t.Errorf("connectionTokenExpired(%q) = %v, want %v", tt.expiresAt, got, tt.want)
			}
		})
	}
}

func TestProbeProviderErrorMessage(t *testing.T) {
	tests := []struct {
		name string
		body string
		want string
	}{
		{name: "nested error message", body: `{"error":{"message":"quota exceeded"}}`, want: "quota exceeded"},
		{name: "plain message", body: `{"message":"boom"}`, want: "boom"},
		{name: "string error", body: `{"error":"bad"}`, want: "bad"},
		{name: "non json", body: `oops`, want: "oops"},
		{name: "empty falls back", body: ``, want: "fallback"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := probeProviderErrorMessage([]byte(tt.body), "fallback"); got != tt.want {
				t.Errorf("probeProviderErrorMessage(%q) = %q, want %q", tt.body, got, tt.want)
			}
		})
	}
}

// Ensure the handler reads the whole body even when the client sends one.
func TestHandleTestConnectionReadsRequestBody(t *testing.T) {
	repo, cleanup := setupProbeTestDB(t)
	defer cleanup()

	if err := repo.CreateProviderConnectionFull("conn-cb2", "codebuddy-cn", "oauth", "CodeBuddy", nil, `{"accessToken":"tok"}`); err != nil {
		t.Fatalf("create connection: %v", err)
	}
	router := setupTestRouter(repo)
	req := httptest.NewRequest(http.MethodPost, "/api/providers/conn-cb2/test", io.NopCloser(strings.NewReader(`{}`)))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
}

func TestHandleTestConnection_EdgeRelay(t *testing.T) {
	repo, cleanup := setupProbeTestDB(t)
	defer cleanup()

	// Create a cloudflare relay proxy pool
	if _, err := repo.RawDB().Exec(`CREATE TABLE IF NOT EXISTS proxyPools (
		id TEXT PRIMARY KEY,
		isActive INTEGER DEFAULT 1,
		testStatus TEXT,
		data TEXT NOT NULL,
		createdAt TEXT NOT NULL,
		updatedAt TEXT NOT NULL
	);`); err != nil {
		t.Fatalf("create table: %v", err)
	}
	poolData := `{"name":"cf-relay","type":"cloudflare","proxyUrl":"https://relay.workers.dev"}`
	if _, err := repo.RawDB().Exec(
		`INSERT INTO proxyPools (id, isActive, testStatus, data, createdAt, updatedAt) VALUES (?, 1, 'active', ?, '2026-01-01T00:00:00Z', '2026-01-01T00:00:00Z')`,
		"pool-cf", poolData,
	); err != nil {
		t.Fatalf("create proxy pool: %v", err)
	}

	// Create connection bound to the pool
	connData := `{"apiKey":"sk-test","providerSpecificData":{"proxyPoolId":"pool-cf","baseUrl":"https://api.openai.com/v1"}}`
	if err := repo.CreateProviderConnectionFull("conn-relay", "openai-compatible-node", "compatible", "OpenAI Relayed", nil, connData); err != nil {
		t.Fatalf("create connection: %v", err)
	}

	origDo := connectionProbeDo
	var gotURL, gotTarget, gotPath string
	connectionProbeDo = func(ctx context.Context, client *http.Client, method, rawURL string, headers map[string]string, body []byte) (int, []byte, error) {
		gotURL = rawURL
		if relay := probeRelayURL(ctx); relay != "" {
			reqHeaders := internalproxy.BuildEdgeRelayHeaders(rawURL, headers)
			gotURL = relay
			gotTarget = reqHeaders["x-relay-target"]
			gotPath = reqHeaders["x-relay-path"]
		}
		return 200, []byte(`{"data":[]}`), nil
	}
	defer func() { connectionProbeDo = origDo }()

	router := setupTestRouter(repo)
	req := httptest.NewRequest(http.MethodPost, "/api/providers/conn-relay/test", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	if gotURL != "https://relay.workers.dev" {
		t.Errorf("expected URL rewritten to relay, got %s", gotURL)
	}
	if gotTarget != "https://api.openai.com" {
		t.Errorf("expected x-relay-target https://api.openai.com, got %s", gotTarget)
	}
	if gotPath != "/v1/models" {
		t.Errorf("expected x-relay-path /v1/models, got %s", gotPath)
	}
}

func TestHandleTestConnection_AntigravityUserinfoFallback(t *testing.T) {
	repo, cleanup := setupProbeTestDB(t)
	defer cleanup()

	connData := `{"accessToken":"valid-token","refreshToken":"refresh-token"}`
	if err := repo.CreateProviderConnectionFull("conn-ag-fallback", "antigravity", "oauth", "Antigravity", nil, connData); err != nil {
		t.Fatalf("create connection: %v", err)
	}

	origDo := connectionProbeDo
	var probedURLs []string
	connectionProbeDo = func(ctx context.Context, client *http.Client, method, rawURL string, headers map[string]string, body []byte) (int, []byte, error) {
		probedURLs = append(probedURLs, rawURL)
		if strings.Contains(rawURL, "loadCodeAssist") {
			// Simulate dial tcp 127.0.0.1:443 connection refused
			return 0, nil, fmt.Errorf("dial tcp 127.0.0.1:443: connect: connection refused")
		}
		if strings.Contains(rawURL, "oauth2/v1/userinfo") {
			return 200, []byte(`{"id":"123","email":"test@example.com"}`), nil
		}
		return 404, nil, fmt.Errorf("not found")
	}
	defer func() { connectionProbeDo = origDo }()

	router := setupTestRouter(repo)
	req := httptest.NewRequest(http.MethodPost, "/api/providers/conn-ag-fallback/test", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	var res map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &res); err != nil {
		t.Fatalf("json unmarshal: %v", err)
	}
	if res["valid"] != true {
		t.Errorf("expected valid true on userinfo fallback, got %v", res)
	}
	if len(probedURLs) < 2 {
		t.Errorf("expected at least 2 probed URLs (code assist then userinfo), got %v", probedURLs)
	}
}
