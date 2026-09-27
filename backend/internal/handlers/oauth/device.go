package oauth

import (
	"bytes"
	json "encoding/json/v2"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"9router/proxy/internal/handlerutil"
)

// deviceProviders lists providers supporting the device-code family.
var deviceProviders = []string{
	"github", "codebuddy-cn", "codebuddy-intl",
}

func deviceSupported(p string) bool {
	for _, v := range deviceProviders {
		if v == p {
			return true
		}
	}
	return false
}

func deviceCanonical(p string) string {
	return p
}

func postJSON(url string, body any, headers map[string]string) (map[string]any, int, error) {
	raw, err := json.Marshal(body)
	if err != nil {
		return nil, 0, err
	}
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(raw))
	if err != nil {
		return nil, 0, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()
	out, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return nil, resp.StatusCode, err
	}
	var data map[string]any
	if len(out) > 0 {
		_ = json.Unmarshal(out, &data)
	}
	return data, resp.StatusCode, nil
}

func postForm(url string, form url.Values, headers map[string]string) (map[string]any, int, error) {
	req, err := http.NewRequest(http.MethodPost, url, strings.NewReader(form.Encode()))
	if err != nil {
		return nil, 0, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()
	out, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return nil, resp.StatusCode, err
	}
	var data map[string]any
	if len(out) > 0 {
		_ = json.Unmarshal(out, &data)
	}
	return data, resp.StatusCode, nil
}

func getJSON(url string, headers map[string]string) (map[string]any, int, error) {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, 0, err
	}
	req.Header.Set("Accept", "application/json")
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()
	out, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return nil, resp.StatusCode, err
	}
	var data map[string]any
	if len(out) > 0 {
		_ = json.Unmarshal(out, &data)
	}
	return data, resp.StatusCode, nil
}

func strVal(m map[string]any, keys ...string) string {
	if m == nil {
		return ""
	}
	for _, k := range keys {
		if s, ok := m[k].(string); ok && s != "" {
			return s
		}
	}
	return ""
}

// HandleDeviceStart begins a device code flow.
// POST /api/oauth/device/start {"provider"}
func (h *OAuthHandler) HandleDeviceStart(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Provider   string `json:"provider"`
		Region     string `json:"region"`
		StartURL   string `json:"startUrl"`
		AuthMethod string `json:"authMethod"`
	}
	if raw, err := io.ReadAll(io.LimitReader(r.Body, 1<<20)); err == nil && len(raw) > 0 {
		_ = json.Unmarshal(raw, &body)
	}
	if !deviceSupported(body.Provider) {
		handlerutil.WriteJSONError(w, http.StatusBadRequest, "unsupported provider for device flow")
		return
	}
	out, err := deviceStart(deviceCanonical(body.Provider), body.Region, body.StartURL, body.AuthMethod)
	if err != nil {
		handlerutil.WriteJSONError(w, http.StatusBadGateway, err.Error())
		return
	}
	out["provider"] = body.Provider
	handlerutil.WriteJSON(w, http.StatusOK, out)
}

func deviceStart(provider, region, startURL, authMethod string) (map[string]any, error) {
	switch provider {
	case "github":
		return githubStart()
	case "codebuddy-cn", "codebuddy-intl":
		return codebuddyStart(provider)
	default:
		return nil, fmt.Errorf("unsupported provider")
	}
}

// --- github: POST https://github.com/login/device/code ---

func githubStart() (map[string]any, error) {
	form := url.Values{
		"client_id": {"Iv1.b507a08c87ecfe98"},
		"scope":     {"read:user"},
	}
	data, status, err := postForm("https://github.com/login/device/code", form, nil)
	if err != nil || status != http.StatusOK {
		return nil, fmt.Errorf("GitHub device flow failed: %v status=%d", err, status)
	}
	return data, nil
}

// --- codebuddy: state nonce -> QR / redirect authUrl ---

func codebuddyVariant(provider string) (base, ua, domain, platform string) {
	if provider == "codebuddy-intl" {
		return "https://www.codebuddy.ai", "IDE/2.63.2 CodeBuddy/2.63.2", "www.codebuddy.ai", "VSCode"
	}
	return "https://copilot.tencent.com", "CLI/2.63.2 CodeBuddy/2.63.2", "copilot.tencent.com", "CLI"
}

func codebuddyStart(provider string) (map[string]any, error) {
	base, ua, domain, platform := codebuddyVariant(provider)
	headers := map[string]string{
		"User-Agent": ua, "X-Requested-With": "XMLHttpRequest",
		"X-Domain": domain, "X-No-Authorization": "true", "X-No-User-Id": "true", "X-Product": "SaaS",
	}
	data, status, err := postJSON(base+"/v2/plugin/auth/state?platform="+platform, map[string]string{}, headers)
	if err != nil || status != http.StatusOK {
		return nil, fmt.Errorf("CodeBuddy state request failed: %v status=%d", err, status)
	}
	var code float64
	if c, ok := data["code"].(float64); ok {
		code = c
	}
	inner, _ := data["data"].(map[string]any)
	state, authURL := strVal(inner, "state"), strVal(inner, "authUrl")
	if code != 0 || state == "" || authURL == "" {
		return nil, fmt.Errorf("CodeBuddy state error: %v", strVal(data, "msg"))
	}
	return map[string]any{
		"device_code": state, "user_code": "",
		"verification_uri": authURL, "verification_uri_complete": authURL,
		"interval": 5, "session": map[string]any{},
	}, nil
}

// HandleDevicePoll polls once; frontend repeats until authorized.
// POST /api/oauth/device/poll {"provider","device_code","session"?}
func (h *OAuthHandler) HandleDevicePoll(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Provider   string         `json:"provider"`
		DeviceCode string         `json:"device_code"`
		Devicecode string         `json:"deviceCode"`
		Session    map[string]any `json:"session"`
	}
	if raw, err := io.ReadAll(io.LimitReader(r.Body, 1<<20)); err == nil && len(raw) > 0 {
		_ = json.Unmarshal(raw, &body)
	}
	if !deviceSupported(body.Provider) {
		handlerutil.WriteJSONError(w, http.StatusBadRequest, "unsupported provider for device flow")
		return
	}
	code := body.DeviceCode
	if code == "" {
		code = body.Devicecode
	}
	if code == "" {
		handlerutil.WriteJSONError(w, http.StatusBadRequest, "missing device_code")
		return
	}
	if body.Session == nil {
		body.Session = map[string]any{}
	}
	tokens, status, errMsg := devicePoll(deviceCanonical(body.Provider), code, body.Session)
	if errMsg != "" {
		handlerutil.WriteJSON(w, http.StatusOK, map[string]any{"status": status, "error": errMsg, "provider": body.Provider})
		return
	}
	if status != "authorized" || strings.TrimSpace(tokens.access) == "" {
		handlerutil.WriteJSON(w, http.StatusOK, map[string]any{"status": "pending", "provider": body.Provider})
		return
	}
	conn := h.saveDeviceConnection(deviceCanonical(body.Provider), tokens)
	handlerutil.WriteJSON(w, http.StatusOK, map[string]any{
		"status": "authorized", "id": conn.id, "connectionId": conn.id,
		"provider": body.Provider, "name": conn.name, "email": conn.email,
	})
}

type deviceTokens struct {
	access, refresh, email, name string
	expiresIn                    int
	extra                        map[string]any
}

func devicePoll(provider, code string, session map[string]any) (deviceTokens, string, string) {
	var t deviceTokens
	var err error
	switch provider {
	case "github":
		t, err = githubPoll(code)
	case "codebuddy-cn", "codebuddy-intl":
		t, err = codebuddyPoll(provider, code)
	default:
		return t, "error", "unsupported provider"
	}
	if err != nil {
		msg := err.Error()
		if strings.Contains(msg, "authorization_pending") || strings.Contains(msg, "slow_down") || strings.Contains(msg, "pending") {
			return t, "pending", ""
		}
		return t, "error", msg
	}
	if t.access == "" {
		return t, "pending", ""
	}
	return t, "authorized", ""
}

type savedConn struct{ id, name, email string }

func (h *OAuthHandler) saveDeviceConnection(provider string, tokens deviceTokens) savedConn {
	name := tokens.name
	if name == "" {
		name = capitalize(provider)
	}
	email := tokens.email
	if email != "" && !strings.Contains(name, "@") {
		name += " (" + email + ")"
	}
	dataMap := map[string]any{
		"accessToken":  tokens.access,
		"refreshToken": tokens.refresh,
		"email":        email,
	}
	if tokens.expiresIn > 0 {
		dataMap["expiresAt"] = time.Now().Add(time.Duration(tokens.expiresIn) * time.Second).UTC().Format(time.RFC3339)
	}
	if len(tokens.extra) > 0 {
		dataMap["providerSpecificData"] = tokens.extra
	}
	dataBytes, _ := json.Marshal(dataMap)
	connID := provider + "-" + shortHash(tokens.access)
	now := currentTimestamp()

	if h.Repo != nil && h.Repo.RawDB() != nil {
		var existing string
		err := h.Repo.RawDB().QueryRow("SELECT data FROM providerConnections WHERE id = ?", connID).Scan(&existing)
		if err == nil && existing != "" {
			_, _ = h.Repo.RawDB().Exec(
				"UPDATE providerConnections SET name = ?, data = ?, updatedAt = ? WHERE id = ?",
				name, string(dataBytes), now, connID,
			)
		} else {
			_, _ = h.Repo.RawDB().Exec(
				"INSERT INTO providerConnections (id, provider, authType, name, isActive, data, createdAt, updatedAt) VALUES (?, ?, 'oauth', ?, 1, ?, ?, ?)",
				connID, provider, name, string(dataBytes), now, now,
			)
		}
	}
	return savedConn{id: connID, name: name, email: email}
}

func githubPoll(code string) (deviceTokens, error) {
	var t deviceTokens
	form := url.Values{
		"client_id": {"Iv1.b507a08c87ecfe98"}, "device_code": {code},
		"grant_type": {"urn:ietf:params:oauth:grant-type:device_code"},
	}
	data, _, err := postForm("https://github.com/login/oauth/access_token", form, nil)
	if err != nil {
		return t, fmt.Errorf("poll_failed: %v", err)
	}
	if e := strVal(data, "error"); e == "authorization_pending" || e == "slow_down" {
		return t, errors.New(e)
	}
	t.access = strVal(data, "access_token")
	t.refresh = strVal(data, "refresh_token")
	if t.access == "" {
		if e := strVal(data, "error"); e != "" {
			return t, fmt.Errorf("%s: %s", e, strVal(data, "error_description"))
		}
		return t, fmt.Errorf("authorization_pending")
	}
	ghH := map[string]string{
		"Authorization": "Bearer " + t.access, "X-GitHub-Api-Version": "2022-11-28",
		"User-Agent": "GitHubCopilotChat/0.26.7",
	}
	var copilot, user map[string]any
	copilot, _, _ = getJSON("https://api.github.com/copilot_internal/v2/token", ghH)
	user, _, _ = getJSON("https://api.github.com/user", ghH)
	t.email = strVal(user, "email")
	t.name = strVal(user, "name", "login")
	t.extra = map[string]any{
		"copilotToken":          strVal(copilot, "token"),
		"copilotTokenExpiresAt": strVal(copilot, "expires_at"),
		"githubUserId":          user["id"], "githubLogin": strVal(user, "login"),
		"githubName": strVal(user, "name"), "githubEmail": strVal(user, "email"),
	}
	return t, nil
}

func codebuddyPoll(provider, state string) (deviceTokens, error) {
	var t deviceTokens
	base, ua, domain := "https://copilot.tencent.com", "CLI/2.63.2 CodeBuddy/2.63.2", "copilot.tencent.com"
	if provider == "codebuddy-intl" {
		base, ua, domain = "https://www.codebuddy.ai", "IDE/2.63.2 CodeBuddy/2.63.2", "www.codebuddy.ai"
	}
	headers := map[string]string{
		"Accept": "application/json", "User-Agent": ua, "X-Requested-With": "XMLHttpRequest",
		"X-Domain": domain, "X-No-Authorization": "true", "X-No-User-Id": "true",
		"X-No-Enterprise-Id": "true", "X-No-Department-Info": "true", "X-Product": "SaaS",
	}
	req, _ := http.NewRequest(http.MethodGet, base+"/v2/plugin/auth/token?state="+url.QueryEscape(state), nil)
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return t, fmt.Errorf("poll_failed: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return t, fmt.Errorf("request_failed")
	}
	out, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	var data map[string]any
	if json.Unmarshal(out, &data) != nil {
		return t, fmt.Errorf("request_failed")
	}
	var code float64
	if c, ok := data["code"].(float64); ok {
		code = c
	}
	inner, _ := data["data"].(map[string]any)
	if code == 0 && strVal(inner, "accessToken") != "" {
		t.access = strVal(inner, "accessToken")
		t.refresh = strVal(inner, "refreshToken")
		return t, nil
	}
	if code == 11217 {
		return t, fmt.Errorf("authorization_pending")
	}
	return t, fmt.Errorf("%s", strVal(data, "msg", "error"))
}
