package proxy

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"9router/proxy/internal/providers"
)

func TestForwardGemini_RelayHeadersForwarded(t *testing.T) {
	var gotTarget, gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotTarget = r.Header.Get("x-relay-target")
		gotPath = r.Header.Get("x-relay-path")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"candidates":[{}]}`))
	}))
	defer srv.Close()

	cfg := &providers.ProviderConfig{
		BaseURL: srv.URL,
		StaticHeaders: map[string]string{
			"x-relay-target": "https://daily-cloudcode-pa.googleapis.com",
			"x-relay-path":   "/v1internal:streamGenerateContent?alt=sse",
		},
	}
	body := `{"model":"gemini-2.5-flash","messages":[{"role":"user","content":"hi"}]}`
	resp, err := ForwardGemini(context.Background(), srv.Client(), cfg, "sk-test", body, true, "proj-123", "gemini-2.5-flash")
	if err != nil {
		t.Fatalf("ForwardGemini unexpected error: %v", err)
	}
	defer resp.Body.Close()

	if gotTarget != "https://daily-cloudcode-pa.googleapis.com" {
		t.Errorf("expected x-relay-target https://daily-cloudcode-pa.googleapis.com, got %q", gotTarget)
	}
	if gotPath != "/v1internal:streamGenerateContent?alt=sse" {
		t.Errorf("expected x-relay-path /v1internal:streamGenerateContent?alt=sse, got %q", gotPath)
	}
}

func TestForwardGemini_RelayUsesRelayURL(t *testing.T) {
	var requestURL string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestURL = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"candidates":[{}]}`))
	}))
	defer srv.Close()

	cfg := &providers.ProviderConfig{
		BaseURL: srv.URL,
		StaticHeaders: map[string]string{
			"x-relay-target": "https://daily-cloudcode-pa.googleapis.com",
			"x-relay-path":   "/v1internal:streamGenerateContent?alt=sse",
		},
	}
	body := `{"model":"gemini-2.5-flash","messages":[{"role":"user","content":"hi"}]}`
	resp, err := ForwardGemini(context.Background(), srv.Client(), cfg, "sk-test", body, true, "proj-123", "gemini-2.5-flash")
	if err != nil {
		t.Fatalf("ForwardGemini unexpected error: %v", err)
	}
	defer resp.Body.Close()

	// When relay headers are present, the request should go to the relay URL root,
	// not have the Gemini action path appended.
	if strings.HasPrefix(requestURL, "/v1internal:") {
		t.Errorf("expected request to go to relay root, got path %q (relay handles path via x-relay-path header)", requestURL)
	}
}

func TestForwardGemini_NoRelayConstructsGeminiURL(t *testing.T) {
	var requestURL string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestURL = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"candidates":[{}]}`))
	}))
	defer srv.Close()

	cfg := &providers.ProviderConfig{
		BaseURL: srv.URL,
	}
	body := `{"model":"gemini-2.5-flash","messages":[{"role":"user","content":"hi"}]}`
	resp, err := ForwardGemini(context.Background(), srv.Client(), cfg, "sk-test", body, true, "", "gemini-2.5-flash")
	if err != nil {
		t.Fatalf("ForwardGemini unexpected error: %v", err)
	}
	defer resp.Body.Close()

	// Without relay headers, the Gemini action path should be appended.
	if !strings.Contains(requestURL, "streamGenerateContent") {
		t.Errorf("expected request path to contain streamGenerateContent, got %q", requestURL)
	}
}

func TestForwardGemini_RelayPreservesOtherStaticHeaders(t *testing.T) {
	var gotHeaders map[string]string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotHeaders = map[string]string{
			"x-custom-header": r.Header.Get("x-custom-header"),
			"x-relay-target":  r.Header.Get("x-relay-target"),
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"candidates":[{}]}`))
	}))
	defer srv.Close()

	cfg := &providers.ProviderConfig{
		BaseURL: srv.URL,
		StaticHeaders: map[string]string{
			"x-relay-target": "https://daily-cloudcode-pa.googleapis.com",
			"x-relay-path":   "/v1internal:streamGenerateContent?alt=sse",
			"x-custom-header": "custom-value",
		},
	}
	body := `{"model":"gemini-2.5-flash","messages":[{"role":"user","content":"hi"}]}`
	resp, err := ForwardGemini(context.Background(), srv.Client(), cfg, "sk-test", body, true, "proj-123", "gemini-2.5-flash")
	if err != nil {
		t.Fatalf("ForwardGemini unexpected error: %v", err)
	}
	defer resp.Body.Close()

	if gotHeaders["x-custom-header"] != "custom-value" {
		t.Errorf("expected x-custom-header custom-value, got %q", gotHeaders["x-custom-header"])
	}
	if gotHeaders["x-relay-target"] != "https://daily-cloudcode-pa.googleapis.com" {
		t.Errorf("expected x-relay-target https://daily-cloudcode-pa.googleapis.com, got %q", gotHeaders["x-relay-target"])
	}
}
