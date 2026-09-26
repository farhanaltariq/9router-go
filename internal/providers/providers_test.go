package providers

import (
	"net/http"
	"testing"
)

func TestKnownProviders_HasExpectedEntries(t *testing.T) {
	wantProviders := []string{
		"openai", "anthropic", "antigravity", "cloudflare-ai",
		"codebuddy-cn", "codebuddy-intl", "codex", "commandcode",
		"github", "nvidia", "ollama",
	}
	for _, p := range wantProviders {
		cfg, ok := KnownProviders[p]
		if !ok {
			t.Errorf("expected KnownProviders to contain %q", p)
			continue
		}
		if cfg.BaseURL == "" {
			t.Errorf("provider %q missing BaseURL", p)
		}
		if cfg.AuthHeader == "" {
			t.Errorf("provider %q missing AuthHeader", p)
		}
		if cfg.AuthScheme == "" {
			t.Errorf("provider %q missing AuthScheme", p)
		}
	}
}

func TestKnownProviders_AuthSchemes(t *testing.T) {
	if cfg := KnownProviders["anthropic"]; cfg.AuthScheme != "raw" || cfg.AuthHeader != "x-api-key" {
		t.Errorf("anthropic expected raw/x-api-key, got %s/%s", cfg.AuthScheme, cfg.AuthHeader)
	}
	if cfg := KnownProviders["openai"]; cfg.AuthScheme != "bearer" || cfg.AuthHeader != "Authorization" {
		t.Errorf("openai expected bearer/Authorization, got %s/%s", cfg.AuthScheme, cfg.AuthHeader)
	}
}

func TestProviderAliasMap_Bidirectional(t *testing.T) {
	cases := map[string]string{
		"ag":   "antigravity",
		"cf":   "cloudflare-ai",
		"cd":   "codebuddy-cn",
		"cbcn": "codebuddy-cn",
		"cbai": "codebuddy-intl",
		"cx":   "codex",
		"cmc":  "commandcode",
		"gh":   "github",
		"nv":   "nvidia",
		"ol":   "ollama",
	}
	for alias, canonical := range cases {
		got, ok := ProviderAliasMap[alias]
		if !ok {
			t.Errorf("expected alias %q to exist", alias)
			continue
		}
		if got != canonical {
			t.Errorf("alias %q: expected %q, got %q", alias, canonical, got)
		}
	}
}

func TestProviderAliasMap_NoSelfCycle(t *testing.T) {
	for alias, canonical := range ProviderAliasMap {
		if alias == canonical {
			t.Errorf("alias %q maps to itself", alias)
		}
	}
}

func TestRetryableStatusCodes(t *testing.T) {
	if !RetryableStatusCodes[http.StatusTooManyRequests] {
		t.Error("expected 429 to be retryable")
	}
	if !RetryableStatusCodes[http.StatusServiceUnavailable] {
		t.Error("expected 503 to be retryable")
	}
	if RetryableStatusCodes[http.StatusInternalServerError] {
		t.Error("500 should not be retryable")
	}
	if RetryableStatusCodes[http.StatusBadRequest] {
		t.Error("400 should not be retryable")
	}
}
