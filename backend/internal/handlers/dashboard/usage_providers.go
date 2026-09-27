package dashboard

import (
	"bytes"
	"context"
	json "encoding/json/v2"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/url"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

// Provider usage fetchers for GET /api/usage/{connectionId}.
// Ports of open-sse/services/usage/*.js, each returning the normalized
// {plan, quotas} shape the dashboard parser consumes. Only live provider
// data — no secrets leave this file except as Bearer headers upstream.

const usageHTTPTimeout = 15 * time.Second

// usageResult is a normalized provider usage response.
type usageResult struct {
	plan    string
	quotas  map[string]any
	message string
	extra   map[string]any
	// bare mirrors upstream handlers (e.g. qoder) that return {message}
	// with NO quotas key on error. Most handlers include quotas:{}.
	bare bool
}

func (r usageResult) toResponse() map[string]any {
	// Mirror upstream: the dashboard hides the quota table whenever a message
	// is set, so only attach a message when there are no quota rows.
	if len(r.quotas) > 0 {
		out := map[string]any{"plan": r.plan, "quotas": r.quotas}
		for k, v := range r.extra {
			out[k] = v
		}
		return out
	}
	if r.message != "" {
		out := map[string]any{"message": r.message}
		if !r.bare {
			out["quotas"] = map[string]any{}
		}
		if r.plan != "" {
			out["plan"] = r.plan
		}
		if len(r.quotas) > 0 {
			out["quotas"] = r.quotas
		}
		return out
	}
	return map[string]any{"plan": r.plan, "quotas": map[string]any{}}
}

// fetchProviderUsage dispatches a live quota fetch for providers with a
// ported usage handler. Returns ok=false for unhandled providers (caller
// falls back to locks/antigravity paths).
func fetchProviderUsage(ctx context.Context, provider string, data map[string]any) (usageResult, bool) {
	accessToken, apiKey, _ := usageCreds(data)
	switch provider {
	case "commandcode":
		return fetchCommandCodeUsage(ctx, apiKey), true
	case "ollama":
		return fetchOllamaUsage(ctx, apiKey), true
	case "codebuddy-intl":
		return fetchCodeBuddyIntlUsage(ctx, accessToken, apiKey), true
	default:
		return usageResult{}, false
	}
}
func usageCreds(data map[string]any) (accessToken, apiKey string, psd map[string]any) {
	if data == nil {
		return "", "", nil
	}
	accessToken, _ = data["accessToken"].(string)
	apiKey, _ = data["apiKey"].(string)
	if psd, ok := data["providerSpecificData"].(map[string]any); ok {
		return accessToken, apiKey, psd
	}
	return accessToken, apiKey, map[string]any{}
}

func psdStr(psd map[string]any, keys ...string) string {
	for _, k := range keys {
		if v, ok := psd[k].(string); ok && strings.TrimSpace(v) != "" {
			return strings.TrimSpace(v)
		}
	}
	return ""
}

func usageGet(ctx context.Context, rawURL string, headers map[string]string) (int, http.Header, []byte, error) {
	return usageDo(ctx, http.MethodGet, rawURL, headers, nil)
}

func usagePost(ctx context.Context, rawURL string, payload map[string]any, headers map[string]string) (int, map[string]any, error) {
	var body []byte
	if payload != nil {
		body, _ = json.Marshal(payload)
	}
	status, _, out, err := usageDo(ctx, http.MethodPost, rawURL, headers, body)
	if err != nil {
		return 0, nil, err
	}
	return status, usageJSON(out), nil
}

func usageDo(ctx context.Context, method, rawURL string, headers map[string]string, body []byte) (int, http.Header, []byte, error) {
	ctx, cancel := context.WithTimeout(ctx, usageHTTPTimeout)
	defer cancel()
	var reader io.Reader
	if body != nil {
		reader = bytes.NewReader(body)
	}
	req, err := http.NewRequestWithContext(ctx, method, rawURL, reader)
	if err != nil {
		return 0, nil, nil, err
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return 0, nil, nil, err
	}
	defer resp.Body.Close()
	out, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	return resp.StatusCode, resp.Header, out, err
}

func usageJSON(out []byte) map[string]any {
	var m map[string]any
	if err := json.Unmarshal(out, &m); err != nil || m == nil {
		return nil
	}
	return m
}

func usageNum(v any, fallback float64) float64 {
	if f, ok := usageFiniteNum(v); ok {
		return f
	}
	return fallback
}

func usageFiniteNum(v any) (float64, bool) {
	switch n := v.(type) {
	case float64:
		if !math.IsNaN(n) && !math.IsInf(n, 0) {
			return n, true
		}
	case float32:
		f := float64(n)
		if !math.IsNaN(f) && !math.IsInf(f, 0) {
			return f, true
		}
	case int:
		return float64(n), true
	case int64:
		return float64(n), true
	case string:
		if s := strings.TrimSpace(n); s != "" {
			if f, err := strconv.ParseFloat(s, 64); err == nil && !math.IsNaN(f) && !math.IsInf(f, 0) {
				return f, true
			}
		}
	case map[string]any:
		// protobuf-json {val: n} envelope.
		if _, ok := n["val"]; ok {
			return usageFiniteNum(n["val"])
		}
	}
	return 0, false
}

func usageStr(v any) string {
	if s, ok := v.(string); ok {
		return s
	}
	return ""
}

// usageResetTime mirrors upstream parseResetTime: unix s/ms, numeric strings,
// ISO strings → RFC3339, else "".
func usageResetTime(v any) string {
	switch t := v.(type) {
	case nil:
		return ""
	case float64:
		if math.IsNaN(t) || math.IsInf(t, 0) {
			return ""
		}
		ms := t
		if ms < 1e12 {
			ms *= 1000
		}
		return time.UnixMilli(int64(ms)).UTC().Format(time.RFC3339)
	case string:
		s := strings.TrimSpace(t)
		if s == "" {
			return ""
		}
		if matched, _ := regexp.MatchString(`^\d+$`, s); matched {
			if n, err := strconv.ParseFloat(s, 64); err == nil {
				return usageResetTime(n)
			}
			return ""
		}
		if tm, err := time.Parse(time.RFC3339, s); err == nil {
			return tm.UTC().Format(time.RFC3339)
		}
		if tm, err := time.Parse("2006-01-02T15:04:05.999999999Z07:00", s); err == nil {
			return tm.UTC().Format(time.RFC3339)
		}
		// Naive datetimes (e.g. CodeBuddy "2026-09-30 23:59:59") parse in the
		// server-local zone — mirroring JS new Date(str) in parseResetTime.
		for _, layout := range []string{"2006-01-02 15:04:05", "2006-01-02"} {
			if tm, err := time.ParseInLocation(layout, s, time.Local); err == nil {
				return tm.UTC().Format(time.RFC3339)
			}
		}
		return ""
	default:
		return ""
	}
}

func usageQuota(used, total float64, resetAt string) map[string]any {
	used = math.Max(0, used)
	total = math.Max(0, total)
	q := map[string]any{"used": used, "total": total, "resetAt": nil, "unlimited": false}
	if resetAt != "" {
		q["resetAt"] = resetAt
	}
	if total > 0 {
		q["remainingPercentage"] = math.Max(0, total-used) / total * 100
	} else {
		q["remainingPercentage"] = 0
	}
	return q
}

func firstNonEmptyStr(vals ...string) string {
	for _, v := range vals {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}

// ---------- commandcode: whoami → credits + subscriptions ----------

var commandCodePlanNames = map[string]string{
	"individual-go": "Go", "individual-goat": "GOAT", "individual-pro": "Pro",
	"individual-pro-v1": "Pro", "individual-provider": "Provider",
	"individual-max": "Max", "individual-ultra": "Ultra", "teams-pro": "Teams Pro",
}

var commandCodePlanCaps = map[string]float64{
	"individual-go": 10, "individual-goat": 70, "individual-pro": 30,
	"individual-pro-v1": 80, "individual-provider": 15,
	"individual-max": 150, "individual-ultra": 300, "teams-pro": 40,
}

func commandCodeWindow(win any) map[string]any {
	m, ok := win.(map[string]any)
	if !ok {
		return nil
	}
	used := usageNum(m["used"], 0)
	total := usageNum(m["cap"], 0)
	if total <= 0 && used <= 0 {
		return nil
	}
	total = math.Max(0, total)
	used = math.Max(0, used)
	return map[string]any{
		"used": used, "total": total, "remaining": math.Max(0, total-used),
		"unlimited": false, "resetAt": usageResetTimeToNil(m["resetAt"]),
	}
}

func usageResetTimeToNil(v any) any {
	if s := usageResetTime(v); s != "" {
		return s
	}
	return nil
}

func fetchCommandCodeUsage(ctx context.Context, apiKey string) usageResult {
	if strings.TrimSpace(apiKey) == "" {
		return usageResult{message: "Command Code API key not available. Add a key to view usage."}
	}
	base := strings.TrimSuffix("https://api.commandcode.ai", "/")
	headers := map[string]string{
		"Authorization": "Bearer " + strings.TrimSpace(apiKey),
		"Accept":        "application/json",
	}
	authErr := usageResult{plan: "Command Code", message: "Command Code authentication failed. Check the API key."}
	status, _, out, err := usageGet(ctx, base+"/alpha/whoami?limits=1", headers)
	if err != nil {
		return usageResult{message: fmt.Sprintf("Command Code error: %v", err)}
	}
	if status == 401 || status == 403 {
		return authErr
	}
	if status < 200 || status >= 300 {
		return usageResult{plan: "Command Code", message: fmt.Sprintf("Command Code usage API error (%d)", status)}
	}
	whoami := usageJSON(out)
	var orgID string
	if org, ok := whoami["org"].(map[string]any); ok {
		orgID, _ = org["id"].(string)
	}
	q := ""
	if orgID != "" {
		q = "?orgId=" + url.QueryEscape(orgID)
	}
	type res struct {
		status int
		out    []byte
		err    error
	}
	creditsCh := make(chan res, 1)
	subsCh := make(chan res, 1)
	go func() {
		s, _, o, e := usageGet(ctx, base+"/alpha/billing/credits"+q, headers)
		creditsCh <- res{s, o, e}
	}()
	go func() {
		s, _, o, e := usageGet(ctx, base+"/alpha/billing/subscriptions"+q, headers)
		subsCh <- res{s, o, e}
	}()
	creditsRes, subsRes := <-creditsCh, <-subsCh
	if creditsRes.err != nil || subsRes.err != nil {
		first := creditsRes.err
		if first == nil {
			first = subsRes.err
		}
		return usageResult{message: fmt.Sprintf("Command Code error: %v", first)}
	}
	for _, s := range []int{creditsRes.status, subsRes.status} {
		if s == 401 || s == 403 {
			return authErr
		}
	}
	if creditsRes.status < 200 || creditsRes.status >= 300 {
		return usageResult{plan: "Command Code", message: fmt.Sprintf("Command Code credits API error (%d)", creditsRes.status)}
	}
	if subsRes.status < 200 || subsRes.status >= 300 {
		return usageResult{plan: "Command Code", message: fmt.Sprintf("Command Code subscriptions API error (%d)", subsRes.status)}
	}
	creditsBody := usageJSON(creditsRes.out)
	subsBody := usageJSON(subsRes.out)
	var planID string
	if d, ok := subsBody["data"].(map[string]any); ok {
		planID, _ = d["planId"].(string)
	}
	plan := commandCodePlanNames[planID]
	if plan == "" {
		plan = firstNonEmptyStr(planID, "Command Code")
	}
	capVal := commandCodePlanCaps[planID]
	var credits map[string]any
	if c, ok := creditsBody["credits"].(map[string]any); ok {
		credits = c
	}
	remaining := usageNum(credits["monthlyCredits"], 0) + usageNum(credits["purchasedCredits"], 0) + usageNum(credits["freeCredits"], 0)
	used, total := 0.0, remaining
	if capVal > 0 {
		used = math.Max(0, capVal-remaining)
		total = capVal
	}
	var periodEnd any
	if d, ok := subsBody["data"].(map[string]any); ok {
		periodEnd = d["currentPeriodEnd"]
	}
	quotas := map[string]any{
		"Credits": map[string]any{
			"used": used, "total": total, "remaining": remaining,
			"unlimited": capVal <= 0, "resetAt": usageResetTimeToNil(periodEnd),
		},
	}
	var windows map[string]any
	if w, ok := creditsBody["windowLimits"].(map[string]any); ok {
		windows = w
	}
	if w := commandCodeWindow(windows["fiveHour"]); w != nil {
		quotas["Session (5h)"] = w
	}
	if w := commandCodeWindow(windows["weekly"]); w != nil {
		quotas["Weekly"] = w
	}
	return usageResult{plan: plan, quotas: quotas}
}

// ---------- ollama cloud: GET /api/usage + POST /api/me ----------

func ollamaRatioQuota(ratio float64) map[string]any {
	ratio = math.Max(0, math.Min(1, ratio))
	used := math.Round(ratio * 100)
	return map[string]any{
		"used": used, "total": 100, "remainingPercentage": 100 - used,
		"resetAt": nil, "unlimited": false,
	}
}

func fetchOllamaUsage(ctx context.Context, apiKey string) usageResult {
	if strings.TrimSpace(apiKey) == "" {
		return usageResult{message: "Ollama Cloud API key not available."}
	}
	headers := map[string]string{
		"Authorization": "Bearer " + strings.TrimSpace(apiKey),
		"Accept":        "application/json",
	}
	status, _, out, err := usageGet(ctx, "https://ollama.com/api/usage", headers)
	if err != nil {
		return usageResult{message: fmt.Sprintf("Ollama Cloud error: %v", err)}
	}
	if status == 401 || status == 403 {
		return usageResult{message: "Ollama Cloud API key invalid or expired."}
	}
	if status < 200 || status >= 300 {
		return usageResult{message: fmt.Sprintf("Ollama Cloud usage API error (%d).", status)}
	}
	data := usageJSON(out)
	if data == nil {
		return usageResult{message: "Ollama Cloud usage response was not JSON."}
	}
	plan := "Ollama Cloud"
	meHeaders := map[string]string{
		"Authorization":  "Bearer " + strings.TrimSpace(apiKey),
		"Accept":         "application/json",
		"Content-Length": "0",
	}
	if s, _, meOut, meErr := usageDo(ctx, http.MethodPost, "https://ollama.com/api/me", meHeaders, nil); meErr == nil && s >= 200 && s < 300 {
		if me := usageJSON(meOut); me != nil {
			if p, _ := me["Plan"].(string); strings.TrimSpace(p) != "" {
				plan = strings.ToUpper(p[:1]) + strings.ToLower(p[1:])
			}
		}
	}
	var limits map[string]any
	if l, ok := data["limits"].(map[string]any); ok {
		limits = l
	}
	sessionQuota, weeklyQuota := map[string]any(nil), map[string]any(nil)
	if s, ok := limits["session"].(map[string]any); ok {
		if u, present := s["usage"]; present {
			if f, ok2 := usageNumOK(u); ok2 {
				sessionQuota = ollamaRatioQuota(f)
			}
		}
	}
	if w, ok := limits["weekly"].(map[string]any); ok {
		if u, present := w["usage"]; present {
			if f, ok2 := usageNumOK(u); ok2 {
				weeklyQuota = ollamaRatioQuota(f)
			}
		}
	}
	if sessionQuota == nil && weeklyQuota == nil {
		return usageResult{plan: plan, message: "Ollama Cloud connected. No usage limits reported.", quotas: map[string]any{}}
	}
	quotas := map[string]any{}
	if sessionQuota != nil {
		quotas["Session (5h)"] = sessionQuota
	}
	if weeklyQuota != nil {
		quotas["Weekly (7d)"] = weeklyQuota
	}
	return usageResult{plan: plan, quotas: quotas}
}

func usageNumOK(v any) (float64, bool) {
	switch n := v.(type) {
	case float64:
		if !math.IsNaN(n) && !math.IsInf(n, 0) {
			return n, true
		}
	case string:
		if s := strings.TrimSpace(n); s != "" {
			if f, err := strconv.ParseFloat(s, 64); err == nil && !math.IsNaN(f) && !math.IsInf(f, 0) {
				return f, true
			}
		}
	}
	return 0, false
}

// ---------- codebuddy-intl: POST billing meter ----------

var codebuddyIntlHeaders = map[string]string{
	"User-Agent":          "IDE/2.108.1 CodeBuddy/2.108.1",
	"X-Product":           "SaaS",
	"X-IDE-Type":          "IDE",
	"X-IDE-Name":          "IDE",
	"X-Requested-With":    "XMLHttpRequest",
	"X-Codebuddy-Request": "1",
	"Content-Type":        "application/json",
	"Accept":              "application/json",
}

func codebuddyNum(precise, plain any) float64 {
	if s, ok := precise.(string); ok && strings.TrimSpace(s) != "" {
		if f, err := strconv.ParseFloat(strings.TrimSpace(s), 64); err == nil && !math.IsNaN(f) && !math.IsInf(f, 0) {
			return f
		}
	}
	return usageNum(plain, 0)
}

func codebuddyCycleEndMs(acc map[string]any) float64 {
	if s := usageResetTime(acc["CycleEndTime"]); s != "" {
		if tm, err := time.Parse(time.RFC3339, s); err == nil {
			return float64(tm.UnixMilli())
		}
	}
	return math.Inf(1)
}

func codebuddyIsRefill(acc map[string]any) bool {
	ce := codebuddyCycleEndMs(acc)
	// DeductionEndTime arrives in unix MILLISECONDS (e.g. 1790429257000);
	// upstream compares it directly against the cycle-end ms.
	de, ok := usageFiniteNum(acc["DeductionEndTime"])
	if math.IsInf(ce, 1) || !ok {
		return false
	}
	return de-ce > float64(2*24*60*60*1000)
}

func codebuddyCadence(acc map[string]any) string {
	start, end := usageResetTime(acc["CycleStartTime"]), usageResetTime(acc["CycleEndTime"])
	if start != "" && end != "" {
		if ts, err1 := time.Parse(time.RFC3339, start); err1 == nil {
			if te, err2 := time.Parse(time.RFC3339, end); err2 == nil {
				days := te.Sub(ts).Hours() / 24
				if days <= 1.5 {
					return "Daily"
				}
				if days <= 10 {
					return "Weekly"
				}
			}
		}
	}
	return "Monthly"
}

func fetchCodeBuddyIntlUsage(ctx context.Context, accessToken, apiKey string) usageResult {
	token := firstNonEmptyStr(accessToken, apiKey)
	if token == "" {
		return usageResult{message: "CodeBuddy (codebuddy-intl) credential not available."}
	}
	headers := map[string]string{"Authorization": "Bearer " + token}
	for k, v := range codebuddyIntlHeaders {
		headers[k] = v
	}
	status, _, out, err := usageDo(ctx, http.MethodPost, "https://www.codebuddy.ai/v2/billing/meter/get-user-resource", headers, []byte("{}"))
	if err != nil {
		return usageResult{message: fmt.Sprintf("CodeBuddy (codebuddy-intl) error: %v", err)}
	}
	if status == 401 || status == 403 {
		return usageResult{message: "CodeBuddy CN credential invalid or expired."}
	}
	if status < 200 || status >= 300 {
		return usageResult{message: fmt.Sprintf("CodeBuddy CN quota API error (%d).", status)}
	}
	body := usageJSON(out)
	if body == nil {
		return usageResult{message: "CodeBuddy (codebuddy-intl) error: invalid JSON"}
	}
	if code := usageNum(body["code"], -1); code != 0 {
		msg, _ := body["msg"].(string)
		if msg == "" {
			msg = "unknown"
		}
		return usageResult{message: fmt.Sprintf("CodeBuddy CN quota error: %s", msg)}
	}
	var data map[string]any
	if d, ok := body["data"].(map[string]any); ok {
		if r, ok := d["Response"].(map[string]any); ok {
			data, _ = r["Data"].(map[string]any)
		}
	}
	var accounts []any
	if a, ok := data["Accounts"].([]any); ok {
		accounts = a
	}
	if len(accounts) == 0 {
		return usageResult{message: "CodeBuddy CN connected. No credit package found."}
	}
	var refills, bonuses []map[string]any
	for _, a := range accounts {
		if acc, ok := a.(map[string]any); ok {
			if codebuddyIsRefill(acc) {
				refills = append(refills, acc)
			} else {
				bonuses = append(bonuses, acc)
			}
		}
	}
	sort.SliceStable(refills, func(i, j int) bool { return codebuddyCycleEndMs(refills[i]) < codebuddyCycleEndMs(refills[j]) })
	sort.SliceStable(bonuses, func(i, j int) bool { return codebuddyCycleEndMs(bonuses[i]) < codebuddyCycleEndMs(bonuses[j]) })
	quotas := map[string]any{}
	seen := map[string]int{}
	for _, acc := range refills {
		base := codebuddyCadence(acc)
		seen[base]++
		name := base
		if seen[base] > 1 {
			name = fmt.Sprintf("%s %d", base, seen[base])
		}
		quotas[name] = map[string]any{
			"used":      codebuddyNum(acc["CycleCapacityUsedPrecise"], acc["CycleCapacityUsed"]),
			"total":     codebuddyNum(acc["CycleCapacitySizePrecise"], acc["CycleCapacitySize"]),
			"resetAt":   usageResetTimeToNil(acc["CycleEndTime"]),
			"unlimited": false, "recurring": true,
		}
	}
	for i, acc := range bonuses {
		quotas[fmt.Sprintf("Bonus Pack %d", i+1)] = map[string]any{
			"used":      codebuddyNum(acc["CapacityUsedPrecise"], acc["CapacityUsed"]),
			"total":     codebuddyNum(acc["CapacitySizePrecise"], acc["CapacitySize"]),
			"resetAt":   usageResetTimeToNil(acc["CycleEndTime"]),
			"unlimited": false, "recurring": false,
		}
	}
	plan := "CodeBuddy"
	base := map[string]any{}
	if len(refills) > 0 {
		base = refills[0]
	} else if len(accounts) > 0 {
		if m, ok := accounts[0].(map[string]any); ok {
			base = m
		}
	}
	if p, _ := base["PackageName"].(string); p != "" {
		plan = p
	} else if p, _ := base["SubProductName"].(string); p != "" {
		plan = p
	}
	return usageResult{plan: plan, quotas: quotas}
}

// ---------------------------------------------------------------------------
// Antigravity (dashboard presentation — parity with open-sse/services/usage/google.js)
// ---------------------------------------------------------------------------
// chat.RefreshAntigravityQuota parses every model and never checks the account
// tier. Upstream getAntigravityUsage instead:
//   - free-tier accounts skip per-model parsing entirely (fetchAvailableModels
//     returns misleading quota info there) and show weekly quotas only;
//   - paid accounts show a curated important-models list on a 1000 base;
//   - both get a best-effort weekly overlay with an exhaustion-reconcile rule.
// Without this, free-tier dashboards render 25 rows stuck at 100%.

var antigravityDashboardBaseURL = "https://cloudcode-pa.googleapis.com"

// antigravityDailyBaseURL serves the usage RPCs. Proven live: PROD
// fetchAvailableModels reports optimistic frac=1.0 while DAILY (the host the
// Next.js usage service uses) omits remainingFraction for exhausted models
// with the real reset time. Dashboard calls must use DAILY; chat routing
// intentionally stays on PROD and is untouched.
var antigravityDailyBaseURL = "https://daily-cloudcode-pa.googleapis.com"

const antigravityDashboardUA = "antigravity/ide/2.11.0 darwin/arm64"

var antigravityImportantModels = map[string]bool{
	"gemini-3.8-flash-high": true, "gemini-3.8-flash-medium": true, "gemini-3.8-flash-low": true,
	"gemini-3.7-flash-high": true, "gemini-3.7-flash-medium": true, "gemini-3.7-flash-low": true,
	"gemini-3.6-flash-high": true, "gemini-3.6-flash-medium": true, "gemini-3.6-flash-low": true,
	"gemini-3.5-flash-low": true, "gemini-3.5-flash-extra-low": true,
	"gemini-pro-agent": true, "gemini-3.1-pro-low": true,
	"claude-sonnet-4-6": true, "claude-opus-4-6-thinking": true,
	"gpt-oss-120b-medium":    true,
	"gemini-3.1-flash-image": true,
}

func antigravityDashboardHeaders(accessToken string) map[string]string {
	return map[string]string{
		"Authorization":    "Bearer " + accessToken,
		"User-Agent":       antigravityDashboardUA,
		"Content-Type":     "application/json",
		"X-Client-Name":    "antigravity",
		"X-Client-Version": "2.11.0",
	}
}

type antigravitySubInfo struct {
	projectID  string
	plan       string
	paidTierID string
}

// fetchAntigravitySubInfo mirrors getAntigravitySubscriptionInfo: one loadCodeAssist
// call reused for projectID + plan + tier detection.
func fetchAntigravitySubInfo(ctx context.Context, accessToken string) antigravitySubInfo {
	var sub antigravitySubInfo
	status, data, err := usagePost(ctx, antigravityDashboardBaseURL+"/v1internal:loadCodeAssist", map[string]any{
		"metadata": map[string]any{"ideType": 9, "platform": 2, "pluginType": 2},
		"mode":     1,
	}, antigravityDashboardHeaders(accessToken))
	if err != nil || status != http.StatusOK || data == nil {
		return sub
	}
	sub.projectID = antigravityProjectID(data["cloudaicompanionProject"])
	if tier, _ := data["currentTier"].(map[string]any); tier != nil {
		sub.plan, _ = tier["name"].(string)
	}
	if paid, _ := data["paidTier"].(map[string]any); paid != nil {
		sub.paidTierID, _ = paid["id"].(string)
	}
	return sub
}

func fetchAntigravityDashboardUsage(ctx context.Context, accessToken, projectID string) usageResult {
	sub := fetchAntigravitySubInfo(ctx, accessToken)
	pid := projectID
	if pid == "" {
		pid = sub.projectID
	}
	plan := sub.plan
	if plan == "" {
		plan = "Unknown"
	}
	headers := antigravityDashboardHeaders(accessToken)
	reqBody := map[string]any{}
	if pid != "" {
		reqBody["project"] = pid
	}
	status, body, err := usagePost(ctx, antigravityDailyBaseURL+"/v1internal:fetchAvailableModels", reqBody, headers)
	if err != nil {
		return usageResult{plan: plan, message: "Antigravity error: " + err.Error()}
	}
	if status == http.StatusForbidden {
		return usageResult{plan: plan, message: "Antigravity quota API access forbidden. Chat may still work."}
	}
	if status == http.StatusUnauthorized {
		return usageResult{plan: plan, message: "Antigravity quota API authentication expired. Chat may still work."}
	}
	if status < 200 || status >= 300 {
		return usageResult{plan: plan, message: fmt.Sprintf("Antigravity error: quota API returned %d.", status)}
	}
	quotas := map[string]any{}
	isFreeTier := sub.paidTierID == "" || sub.paidTierID == "free-tier"
	if !isFreeTier {
		if models, _ := body["models"].(map[string]any); models != nil {
			for modelKey, infoRaw := range models {
				info, _ := infoRaw.(map[string]any)
				if info == nil {
					continue
				}
				qi, _ := info["quotaInfo"].(map[string]any)
				if qi == nil {
					continue
				}
				if internal, _ := info["isInternal"].(bool); internal || !antigravityImportantModels[modelKey] {
					continue
				}
				frac := usageNum(qi["remainingFraction"], 0)
				total := 1000.0
				remaining := math.Round(total * frac)
				used := math.Max(0, total-remaining)
				dn, _ := info["displayName"].(string)
				if dn == "" {
					dn = modelKey
				}
				quotas[modelKey] = map[string]any{
					"used": used, "total": total,
					"resetAt":             usageResetTime(qi["resetTime"]),
					"remainingPercentage": frac * 100,
					"unlimited":           false,
					"displayName":         dn,
				}
			}
		}
	}
	// Best-effort weekly overlay — never breaks per-model results.
	// Hits DAILY retrieveUserQuotaSummary like upstream (PROD returns a
	// different weekly view) and parses weekly-only buckets like upstream
	// (no session rows — upstream shows 19 rows, not 21).
	if weekly := fetchAntigravityDashboardWeekly(ctx, accessToken, pid); len(weekly) > 0 {
		for key, wq := range weekly {
			if key == "gemini_weekly" {
				antigravityReconcileFamily(quotas, "gemini-", "image", wq)
			} else if key == "claude_gpt_weekly" {
				antigravityReconcileFamily(quotas, "claude-", "", wq)
			}
			quotas[key] = wq
		}

		// Reconcile gemini_session if all gemini models are exhausted (upstream google.js parity)
		if s, ok := quotas["gemini_session"].(map[string]any); ok {
			allGeminiExhausted := true
			var geminiCount int
			var maxResetAt string
			for k, v := range quotas {
				if strings.HasPrefix(k, "gemini-") && !strings.Contains(k, "image") {
					geminiCount++
					if vm, ok := v.(map[string]any); ok {
						if rem, ok := vm["remainingPercentage"].(float64); ok && rem > 0 {
							allGeminiExhausted = false
						}
						if rAt, ok := vm["resetAt"].(string); ok && rAt != "" {
							if maxResetAt == "" || rAt > maxResetAt {
								maxResetAt = rAt
							}
						}
					}
				}
			}
			if geminiCount > 0 && allGeminiExhausted {
				s["used"] = s["total"]
				s["remainingPercentage"] = 0.0
				if maxResetAt != "" {
					s["resetAt"] = maxResetAt
				}
			}
		}
	}
	return usageResult{plan: plan, quotas: quotas}
}

// fetchAntigravityDashboardWeekly mirrors antigravity-weekly.js: DAILY
// retrieveUserQuotaSummary, weekly-only buckets, first match per family wins.
func fetchAntigravityDashboardWeekly(ctx context.Context, accessToken, projectID string) map[string]any {
	reqBody := map[string]any{}
	if projectID != "" {
		reqBody["project"] = projectID
	}
	status, data, err := usagePost(ctx, antigravityDailyBaseURL+"/v1internal:retrieveUserQuotaSummary", reqBody, antigravityDashboardHeaders(accessToken))
	if err != nil || status < 200 || status >= 300 || data == nil {
		return nil
	}
	var groups []any
	if g, ok := data["groups"].([]any); ok && len(g) > 0 {
		groups = g
	} else if qs, ok := data["quotaSummary"].(map[string]any); ok {
		groups, _ = qs["groups"].([]any)
	}
	if len(groups) == 0 {
		return nil
	}
	type familyTarget struct {
		key         string
		displayName string
	}
	type familyConfig struct {
		pattern string
		weekly  familyTarget
		session familyTarget
	}
	configs := []familyConfig{
		{
			pattern: "gemini",
			weekly:  familyTarget{"gemini_weekly", "Gemini (Weekly)"},
			session: familyTarget{"gemini_session", "Gemini (5h)"},
		},
		{
			pattern: "claude",
			weekly:  familyTarget{"claude_gpt_weekly", "Claude & GPT (Weekly)"},
			session: familyTarget{"claude_gpt_session", "Claude & GPT (5h)"},
		},
	}
	result := map[string]any{}
	for _, gRaw := range groups {
		g, _ := gRaw.(map[string]any)
		if g == nil {
			continue
		}
		gName, _ := g["displayName"].(string)
		gNameLower := strings.ToLower(gName)
		buckets, _ := g["buckets"].([]any)
		for _, bRaw := range buckets {
			b, _ := bRaw.(map[string]any)
			if b == nil {
				continue
			}
			windowType := strings.ToLower(usageStr(b["window"]))
			bucketText := strings.ToLower(usageStr(b["bucketId"]) + " " + usageStr(b["displayName"]))
			isWeekly := windowType == "weekly" || strings.Contains(bucketText, "weekly")
			isSession := windowType == "5h" || strings.Contains(bucketText, "five hour") || strings.Contains(bucketText, "5h") || strings.Contains(bucketText, "daily") || windowType == "daily"
			if !isWeekly && !isSession {
				continue
			}
			disabled, _ := b["disabled"].(bool)
			if disabled && isWeekly {
				continue
			}
			var frac float64
			if disabled {
				frac = 0
			} else {
				var ok bool
				frac, ok = usageFiniteNum(b["remainingFraction"])
				if !ok {
					continue
				}
			}
			for _, cfg := range configs {
				if strings.Contains(gNameLower, cfg.pattern) || (cfg.pattern == "claude" && strings.Contains(gNameLower, "gpt")) {
					target := cfg.session
					if isWeekly {
						target = cfg.weekly
					}
					if _, exists := result[target.key]; exists {
						break
					}
					total := 1000.0
					remaining := math.Round(total * frac)
					used := math.Max(0, total-remaining)
					result[target.key] = map[string]any{
						"used":                used,
						"total":               total,
						"resetAt":             usageResetTime(b["resetTime"]),
						"remainingPercentage": frac * 100,
						"unlimited":           false,
						"displayName":         target.displayName,
					}
					break
				}
			}
		}
	}
	return result
}

// antigravityProjectID mirrors chat.extractProjectID (unexported there):
// cloudaicompanionProject arrives as a string id or an {id} object.
func antigravityProjectID(val any) string {
	if s, ok := val.(string); ok {
		return strings.TrimSpace(s)
	}
	if m, ok := val.(map[string]any); ok {
		if id, _ := m["id"].(string); id != "" {
			return strings.TrimSpace(id)
		}
	}
	return ""
}

// antigravityReconcileFamily mirrors the upstream reconcile: when every model of
// a family reads exhausted but weekly still claims availability, weekly is
// forced to exhausted (Free Starter tier misreports remainingFraction: 1).
func antigravityReconcileFamily(quotas map[string]any, prefix, exclude string, wqRaw any) {
	wq, _ := wqRaw.(map[string]any)
	if wq == nil {
		return
	}
	var family []map[string]any
	var maxReset string
	for k, v := range quotas {
		if !strings.HasPrefix(k, prefix) {
			continue
		}
		if exclude != "" && strings.Contains(k, exclude) {
			continue
		}
		m, _ := v.(map[string]any)
		if m == nil {
			continue
		}
		family = append(family, m)
		if rs, _ := m["resetAt"].(string); rs > maxReset {
			maxReset = rs
		}
	}
	if len(family) == 0 {
		return
	}
	for _, m := range family {
		if p, _ := m["remainingPercentage"].(float64); p != 0 {
			return
		}
	}
	if pct, _ := wq["remainingPercentage"].(float64); pct > 0 {
		total, _ := wq["total"].(float64)
		wq["used"] = total
		wq["remainingPercentage"] = 0.0
		if maxReset != "" {
			wq["resetAt"] = maxReset
		}
	}
}
