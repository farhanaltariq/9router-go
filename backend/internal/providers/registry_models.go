package providers

import "strings"

var ProviderModels = map[string][]string{
	"ag":             {"gemini-2.5-flash", "gemini-2.5-flash-lite", "gemini-2.5-pro", "gemini-2.5-flash-thinking", "gemini-3.1-pro-high", "gemini-3.1-flash-lite", "gemini-3.5-flash-lite", "gemini-3.8-flash-high", "gemini-3.8-flash-medium", "gemini-3.8-flash-low", "gemini-3.8-flash", "gemini-3.7-flash-high", "gemini-3.7-flash-medium", "gemini-3.7-flash-low", "gemini-3.6-flash-high", "gemini-3.6-flash-medium", "gemini-3.6-flash-low", "gemini-3.5-flash-high", "gemini-3-flash-agent", "gemini-3.5-flash-low", "gemini-3.5-flash-extra-low", "gemini-pro-agent", "gemini-3.1-pro-low", "claude-sonnet-4-6", "claude-opus-4-6-thinking", "gpt-oss-120b-medium", "gemini-3-flash", "gemini-3.1-flash-image"},
	"antigravity":    {"gemini-2.5-flash", "gemini-2.5-flash-lite", "gemini-2.5-pro", "gemini-2.5-flash-thinking", "gemini-3.1-pro-high", "gemini-3.1-flash-lite", "gemini-3.5-flash-lite", "gemini-3.8-flash-high", "gemini-3.8-flash-medium", "gemini-3.8-flash-low", "gemini-3.8-flash", "gemini-3.7-flash-high", "gemini-3.7-flash-medium", "gemini-3.7-flash-low", "gemini-3.6-flash-high", "gemini-3.6-flash-medium", "gemini-3.6-flash-low", "gemini-3.5-flash-high", "gemini-3-flash-agent", "gemini-3.5-flash-low", "gemini-3.5-flash-extra-low", "gemini-pro-agent", "gemini-3.1-pro-low", "claude-sonnet-4-6", "claude-opus-4-6-thinking", "gpt-oss-120b-medium", "gemini-3-flash", "gemini-3.1-flash-image"},
	"cbai":           {"glm-5.2", "glm-5.1", "glm-5.0", "glm-5.0-turbo", "glm-5v-turbo", "glm-4.7", "minimax-m3", "minimax-m2.7", "kimi-k2.7", "kimi-k2.6", "kimi-k2.5", "hy3-preview", "deepseek-v4-pro", "deepseek-v4.1-flash", "deepseek-v3-2-volc"},
	"cbcn":           {"glm-5.2", "glm-5.1", "glm-5v-turbo", "minimax-m3", "kimi-k2.7", "kimi-k2.6", "hy3", "hy4-preview", "glm-5.3", "glm-5.3-flash", "kimi-k3-1", "deepseek-v4-pro", "deepseek-v4.1-flash"},
	"cloudflare-ai":  {"@cf/meta/llama-3.2-1b-instruct", "@cf/meta/llama-3.2-3b-instruct", "@cf/meta/llama-3.1-8b-instruct-fp8-fast", "@cf/meta/llama-3.1-8b-instruct-awq", "@cf/mistralai/mistral-small-3.1-24b-instruct", "@cf/meta/llama-3.1-70b-instruct-fp8-fast", "@cf/meta/llama-3.3-70b-instruct-fp8-fast", "@cf/deepseek-ai/deepseek-r1-distill-qwen-32b", "@cf/moonshotai/kimi-k2.5", "@cf/moonshotai/kimi-k2.6", "@cf/zai-org/glm-4.7-flash", "@cf/qwen/qwq-32b", "@cf/qwen/qwen2.5-coder-32b-instruct", "@cf/black-forest-labs/flux-2-klein-9b", "@cf/black-forest-labs/flux-2-klein-4b", "@cf/black-forest-labs/flux-2-dev", "@cf/leonardo/lucid-origin", "@cf/leonardo/phoenix-1.0", "@cf/black-forest-labs/flux-1-schnell", "@cf/bytedance/stable-diffusion-xl-lightning", "@cf/lykon/dreamshaper-8-lcm", "@cf/runwayml/stable-diffusion-v1-5-img2img", "@cf/runwayml/stable-diffusion-v1-5-inpainting", "@cf/stabilityai/stable-diffusion-xl-base-1.0"},
	"cf":             {"@cf/meta/llama-3.2-1b-instruct", "@cf/meta/llama-3.2-3b-instruct", "@cf/meta/llama-3.1-8b-instruct-fp8-fast", "@cf/meta/llama-3.1-8b-instruct-awq", "@cf/mistralai/mistral-small-3.1-24b-instruct", "@cf/meta/llama-3.1-70b-instruct-fp8-fast", "@cf/meta/llama-3.3-70b-instruct-fp8-fast", "@cf/deepseek-ai/deepseek-r1-distill-qwen-32b", "@cf/moonshotai/kimi-k2.5", "@cf/moonshotai/kimi-k2.6", "@cf/zai-org/glm-4.7-flash", "@cf/qwen/qwq-32b", "@cf/qwen/qwen2.5-coder-32b-instruct", "@cf/black-forest-labs/flux-2-klein-9b", "@cf/black-forest-labs/flux-2-klein-4b", "@cf/black-forest-labs/flux-2-dev", "@cf/leonardo/lucid-origin", "@cf/leonardo/phoenix-1.0", "@cf/black-forest-labs/flux-1-schnell", "@cf/bytedance/stable-diffusion-xl-lightning", "@cf/lykon/dreamshaper-8-lcm", "@cf/runwayml/stable-diffusion-v1-5-img2img", "@cf/runwayml/stable-diffusion-v1-5-inpainting", "@cf/stabilityai/stable-diffusion-xl-base-1.0"},
	"codebuddy-cn":   {"glm-5.2", "glm-5.1", "glm-5v-turbo", "minimax-m3", "kimi-k2.7", "kimi-k2.6", "hy3", "hy4-preview", "glm-5.3", "glm-5.3-flash", "kimi-k3-1", "deepseek-v4-pro", "deepseek-v4.1-flash"},
	"cd":             {"glm-5.2", "glm-5.1", "glm-5v-turbo", "minimax-m3", "kimi-k2.7", "kimi-k2.6", "hy3", "hy4-preview", "glm-5.3", "glm-5.3-flash", "kimi-k3-1", "deepseek-v4-pro", "deepseek-v4.1-flash"},
	"codebuddy-intl": {"glm-5.2", "glm-5.1", "glm-5.0", "glm-5.0-turbo", "glm-5v-turbo", "glm-4.7", "minimax-m3", "minimax-m2.7", "kimi-k2.7", "kimi-k2.6", "kimi-k2.5", "hy3-preview", "deepseek-v4-pro", "deepseek-v4.1-flash", "deepseek-v3-2-volc"},
	"codex":          {"codex-auto-review", "gpt-6-astra", "gpt-5.6-sol", "gpt-5.6-sol-review", "gpt-5.6-terra", "gpt-5.6-terra-review", "gpt-5.6-luna", "gpt-5.6-luna-review", "gpt-5.5", "gpt-5.5-review", "gpt-5.4", "gpt-5.4-review", "gpt-5.4-mini", "gpt-5.4-mini-review", "gpt-5.3-codex-spark", "gpt-5.3-codex-spark-review", "gpt-5.6-sol-image", "gpt-5.6-terra-image", "gpt-5.6-luna-image", "gpt-5.5-image", "gpt-5.4-image", "gpt-5.3-image"},
	"cx":             {"gpt-6-astra", "gpt-5.6-sol", "gpt-5.6-sol-review", "gpt-5.6-terra", "gpt-5.6-terra-review", "gpt-5.6-luna", "gpt-5.6-luna-review", "gpt-5.5", "gpt-5.5-review", "gpt-5.4", "gpt-5.4-review", "gpt-5.4-mini", "gpt-5.4-mini-review", "gpt-5.3-codex-spark", "gpt-5.3-codex-spark-review", "gpt-5.6-sol-image", "gpt-5.6-terra-image", "gpt-5.6-luna-image", "gpt-5.5-image", "gpt-5.4-image", "gpt-5.3-image"},
	"commandcode":    {"deepseek/deepseek-v4-pro", "deepseek/deepseek-v4-flash", "moonshotai/Kimi-K2.6", "moonshotai/Kimi-K2.5", "zai-org/GLM-5.1", "zai-org/GLM-5", "MiniMaxAI/MiniMax-M2.7", "MiniMaxAI/MiniMax-M2.5", "Qwen/Qwen3.6-Max-Preview", "Qwen/Qwen3.6-Plus", "stepfun/Step-3.5-Flash"},
	"cmc":            {"deepseek/deepseek-v4-pro", "deepseek/deepseek-v4-flash", "moonshotai/Kimi-K2.6", "moonshotai/Kimi-K2.5", "zai-org/GLM-5.1", "zai-org/GLM-5", "MiniMaxAI/MiniMax-M2.7", "MiniMaxAI/MiniMax-M2.5", "Qwen/Qwen3.6-Max-Preview", "Qwen/Qwen3.6-Plus", "stepfun/Step-3.5-Flash"},
	"gh":             {"gpt-5.2", "gpt-5.2-codex", "gpt-5.3-codex", "gpt-5.4", "gpt-5.4-mini", "claude-haiku-4.5", "claude-opus-4.5", "claude-sonnet-4.5", "claude-sonnet-4.6", "claude-opus-4.6", "claude-opus-4.7", "gemini-2.5-pro", "gemini-3-flash-preview", "gemini-3.1-pro-preview", "grok-code-fast-1", "oswe-vscode-prime", "goldeneye-free-auto", "text-embedding-3-small", "text-embedding-3-large"},
	"github":         {"gpt-5.2", "gpt-5.2-codex", "gpt-5.3-codex", "gpt-5.4", "gpt-5.4-mini", "claude-haiku-4.5", "claude-opus-4.5", "claude-sonnet-4.5", "claude-sonnet-4.6", "claude-opus-4.6", "claude-opus-4.7", "gemini-2.5-pro", "gemini-3-flash-preview", "gemini-3.1-pro-preview", "grok-code-fast-1", "oswe-vscode-prime", "goldeneye-free-auto", "text-embedding-3-small", "text-embedding-3-large"},
	"nv":             {"minimaxai/minimax-m2.7", "minimaxai/minimax-m3", "z-ai/glm-5.2", "deepseek-ai/deepseek-v4-pro", "deepseek-ai/deepseek-v4-flash", "moonshotai/kimi-k2.6", "nvidia/nemotron-3-ultra-550b-a55b", "nvidia/nv-embedqa-e5-v5", "nvidia/parakeet-ctc-1.1b-asr", "fastpitch", "tacotron2"},
	"nvidia":         {"minimaxai/minimax-m2.7", "minimaxai/minimax-m3", "z-ai/glm-5.2", "deepseek-ai/deepseek-v4-pro", "deepseek-ai/deepseek-v4-flash", "moonshotai/kimi-k2.6", "nvidia/nemotron-3-ultra-550b-a55b", "nvidia/nv-embedqa-e5-v5", "nvidia/parakeet-ctc-1.1b-asr", "fastpitch", "tacotron2"},
	"ol":             {"gpt-oss:120b", "kimi-k2.5", "glm-5", "minimax-m2.5", "glm-4.7-flash", "qwen3.5", "minimax-m3", "deepseek-v4.1-flash:cloud"},
	"ollama":         {"gpt-oss:120b", "kimi-k2.5", "glm-5", "minimax-m2.5", "glm-4.7-flash", "qwen3.5", "minimax-m3", "deepseek-v4.1-flash:cloud"},
	"oc":             {"muse-spark-1.2-contributor-free", "muse-spark-1.3-contributor-free"},
	"opencode":       {"muse-spark-1.2-contributor-free", "muse-spark-1.3-contributor-free"},
}

// GetProviderModels returns the list of models for a provider or alias.
func GetProviderModels(providerOrAlias string) []string {
	if models, ok := ProviderModels[providerOrAlias]; ok {
		return models
	}
	canonical := ResolveAlias(providerOrAlias)
	if models, ok := ProviderModels[canonical]; ok {
		return models
	}
	lower := strings.ToLower(providerOrAlias)
	if models, ok := ProviderModels[lower]; ok {
		return models
	}
	return nil
}
