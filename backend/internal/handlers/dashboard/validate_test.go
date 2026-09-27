package dashboard

import (
	"context"
	json "encoding/json/v2"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

type capturedProbe struct {
	method  string
	url     string
	headers http.Header
	body    string
}

func withProbeStub(t *testing.T, respond func(call int, p capturedProbe) (int, []byte, error)) *[]capturedProbe {
	t.Helper()
	orig := validateProbeDo
	var calls []capturedProbe
	validateProbeDo = func(_ context.Context, method, rawURL string, headers map[string]string, body []byte) (int, []byte, error) {
		h := http.Header{}
		for k, v := range headers {
			h.Set(k, v)
		}
		p := capturedProbe{method: method, url: rawURL, headers: h, body: string(body)}
		calls = append(calls, p)
		return respond(len(calls)-1, p)
	}
	t.Cleanup(func() { validateProbeDo = orig })
	return &calls
}

func staticProbeStub(t *testing.T, statuses ...int) *[]capturedProbe {
	t.Helper()
	return withProbeStub(t, func(call int, _ capturedProbe) (int, []byte, error) {
		if call >= len(statuses) {
			return statuses[len(statuses)-1], nil, nil
		}
		return statuses[call], nil, nil
	})
}

func errorMessage(t *testing.T, out map[string]any) string {
	t.Helper()
	if out == nil {
		return ""
	}
	if s, ok := out["error"].(string); ok {
		return s
	}
	if errMap, ok := out["error"].(map[string]any); ok {
		if s, ok := errMap["message"].(string); ok {
			return s
		}
	}
	return ""
}

func postValidate(t *testing.T, body string) (*httptest.ResponseRecorder, map[string]any) {
	t.Helper()
	h := NewDashboardHandler(nil)
	req := httptest.NewRequest(http.MethodPost, "/api/providers/validate", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	h.HandleValidateProvider(rec, req)

	var out map[string]any
	if rec.Body.Len() > 0 {
		_ = json.Unmarshal(rec.Body.Bytes(), &out)
	}
	return rec, out
}

func TestHandleValidateProvider_RequiresProviderAndKey(t *testing.T) {
	calls := staticProbeStub(t, http.StatusOK)

	rec, out := postValidate(t, `{}`)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("empty body: expected 400, got %d (%s)", rec.Code, rec.Body.String())
	}
	if got := errorMessage(t, out); got != "Provider and API key required" {
		t.Errorf("empty body: unexpected error %q", got)
	}

	rec, _ = postValidate(t, `{"provider":"nvidia"}`)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("missing key: expected 400, got %d", rec.Code)
	}

	if len(*calls) != 0 {
		t.Errorf("expected no upstream probe for invalid requests, got %d", len(*calls))
	}
}

func TestHandleValidateProvider_UnknownProviderIsUnsupported(t *testing.T) {
	calls := staticProbeStub(t, http.StatusOK)

	rec, out := postValidate(t, `{"provider":"no-such-provider","apiKey":"k"}`)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d (%s)", rec.Code, rec.Body.String())
	}
	if got := errorMessage(t, out); got != "Provider validation not supported" {
		t.Errorf("unexpected error %q", got)
	}
	if len(*calls) != 0 {
		t.Errorf("expected no upstream probe for unknown provider, got %d", len(*calls))
	}
}

func TestHandleValidateProvider_NvidiaProbesRegistryModelsEndpoint(t *testing.T) {
	calls := staticProbeStub(t, http.StatusOK)

	rec, out := postValidate(t, `{"provider":"nvidia","apiKey":"sk-test"}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d (%s)", rec.Code, rec.Body.String())
	}
	if out["valid"] != true {
		t.Errorf("expected valid=true, got %v", out["valid"])
	}
	if out["supported"] != true {
		t.Errorf("expected supported=true, got %v", out["supported"])
	}
	if out["error"] != nil {
		t.Errorf("expected null error, got %v", out["error"])
	}
	if len(*calls) != 1 {
		t.Fatalf("expected exactly one probe, got %d", len(*calls))
	}
	probe := (*calls)[0]
	if probe.method != http.MethodGet || probe.url != "https://integrate.api.nvidia.com/v1/models" {
		t.Errorf("unexpected probe %s %s", probe.method, probe.url)
	}
	if got := probe.headers.Get("Authorization"); got != "Bearer sk-test" {
		t.Errorf("expected bearer auth header, got %q", got)
	}
}

func TestHandleValidateProvider_AliasResolvesToCanonicalProvider(t *testing.T) {
	calls := staticProbeStub(t, http.StatusOK)

	rec, _ := postValidate(t, `{"provider":"nv","apiKey":"sk-test"}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d (%s)", rec.Code, rec.Body.String())
	}
	if len(*calls) != 1 || (*calls)[0].url != "https://integrate.api.nvidia.com/v1/models" {
		t.Fatalf("alias nv did not resolve to nvidia: %+v", *calls)
	}
}

func TestHandleValidateProvider_InvalidKeyIsAValidResponse(t *testing.T) {
	staticProbeStub(t, http.StatusUnauthorized)

	rec, out := postValidate(t, `{"provider":"nvidia","apiKey":"nope"}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 with valid=false, got %d", rec.Code)
	}
	if out["valid"] != false {
		t.Errorf("expected valid=false, got %v", out["valid"])
	}
	if out["error"] != "Invalid API key" {
		t.Errorf("expected 'Invalid API key', got %v", out["error"])
	}
}
