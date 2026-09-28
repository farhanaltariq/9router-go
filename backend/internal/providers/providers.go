package providers

import (
	"strings"
)

// ProviderConfig holds upstream configuration for a provider.
type ProviderConfig struct {
	BaseURL       string            `json:"baseUrl"`
	AuthHeader    string            `json:"authHeader"`
	AuthScheme    string            `json:"authScheme"` // "bearer" | "raw"
	NoAuth        bool              `json:"noAuth,omitempty"`
	DefaultAPIKey string            `json:"defaultApiKey,omitempty"`
	StaticHeaders map[string]string `json:"staticHeaders,omitempty"`
	Format        string            `json:"format,omitempty"` // "gemini-native" etc.
	ImageURL      string            `json:"imageUrl,omitempty"`
	TTSURL        string            `json:"ttsUrl,omitempty"`
	STTURL        string            `json:"sttUrl,omitempty"`
	VideoURL      string            `json:"videoUrl,omitempty"`
	VoicesURL     string            `json:"voicesUrl,omitempty"`
	SystemoneURL  string            `json:"systemoneUrl,omitempty"`
	FetchURL      string            `json:"fetchUrl,omitempty"`
	FetchMethod   string            `json:"fetchMethod,omitempty"`
}

// IsGeminiNative returns true if the provider uses Google's native RPC format.
func (p *ProviderConfig) IsGeminiNative() bool {
	return p.Format == "gemini-native"
}

// IsGeminiOpenAICompat returns true if the provider uses Google's OpenAI-compatible endpoint.
func (p *ProviderConfig) IsGeminiOpenAICompat() bool {
	return strings.Contains(p.BaseURL, "generativelanguage.googleapis.com") && strings.HasSuffix(p.BaseURL, "/chat/completions")
}

// KnownProviders maps curated provider IDs to their upstream configuration.
var KnownProviders = map[string]ProviderConfig{
	"openai": {
		BaseURL:    "https://api.openai.com/v1/chat/completions",
		AuthHeader: "Authorization",
		AuthScheme: "bearer",
	},
	"anthropic": {
		BaseURL:    "https://api.anthropic.com/v1/messages",
		AuthHeader: "x-api-key",
		AuthScheme: "raw",
	},
	"antigravity": {
		BaseURL:    "https://daily-cloudcode-pa.googleapis.com",
		AuthHeader: "Authorization",
		AuthScheme: "bearer",
		Format:     "gemini-native",
		ImageURL:   "https://daily-cloudcode-pa.googleapis.com/v1internal:generateContent",
	},
	"cloudflare-ai": {
		BaseURL:    "https://api.cloudflare.com/client/v4/accounts/%s/ai/v1/chat/completions",
		AuthHeader: "Authorization",
		AuthScheme: "bearer",
	},
	"codebuddy-cn": {
		BaseURL:    "https://copilot.tencent.com/v2/plugin/chat/completions",
		AuthHeader: "Authorization",
		AuthScheme: "bearer",
	},
	"codebuddy-intl": {
		BaseURL:    "https://www.codebuddy.ai/v2/plugin/chat/completions",
		AuthHeader: "Authorization",
		AuthScheme: "bearer",
	},
	"codex": {
		BaseURL:    "https://chatgpt.com/backend-api/codex/responses",
		AuthHeader: "Authorization",
		AuthScheme: "bearer",
		ImageURL:   "https://chatgpt.com/backend-api/codex/responses",
	},
	"commandcode": {
		BaseURL:    "https://api.commandcode.ai/alpha/generate",
		AuthHeader: "Authorization",
		AuthScheme: "bearer",
	},
	"github": {
		BaseURL:    "https://models.inference.ai.azure.com/chat/completions",
		AuthHeader: "Authorization",
		AuthScheme: "bearer",
	},
	"nvidia": {
		BaseURL:    "https://integrate.api.nvidia.com/v1/chat/completions",
		AuthHeader: "Authorization",
		AuthScheme: "bearer",
	},
	"ollama": {
		BaseURL:    "https://ollama.com/api/chat",
		AuthHeader: "Authorization",
		AuthScheme: "bearer",
	},
	"opencode": {
		BaseURL:       "https://opencode.ai/zen/v1/chat/completions",
		AuthHeader:    "Authorization",
		AuthScheme:    "bearer",
		DefaultAPIKey: "public",
		NoAuth:        true,
		StaticHeaders: map[string]string{"x-opencode-client": "desktop", "User-Agent": "opencode/1.18.31"},
	},
}

// RetryableStatusCodes defines which HTTP status codes trigger account fallback.
var RetryableStatusCodes = map[int]bool{
	401: true,
	403: true,
	429: true,
	502: true,
	503: true,
	504: true,
}
