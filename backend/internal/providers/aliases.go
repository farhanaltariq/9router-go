package providers

// ProviderAliasMap maps short aliases to canonical provider IDs.
var ProviderAliasMap = map[string]string{
	"ag":   "antigravity",
	"cf":   "cloudflare-ai",
	"cd":   "codebuddy-cn",
	"cbcn": "codebuddy-cn",
	"cbai": "codebuddy-intl",
	"cx":   "codex",
	"cmc":  "commandcode",
	"gh":   "github",
	"nv":   "nvidia",
	"oc":   "opencode",
	"ol":   "ollama",
}

// ResolveAlias returns the canonical provider ID for an alias, or the alias itself if not found.
func ResolveAlias(alias string) string {
	if canonical, ok := ProviderAliasMap[alias]; ok {
		return canonical
	}
	return alias
}

// ProviderToAliasMap maps canonical provider IDs to their primary short alias.
var ProviderToAliasMap = map[string]string{}

func init() {
	for alias, provider := range ProviderAliasMap {
		if _, exists := ProviderToAliasMap[provider]; !exists {
			ProviderToAliasMap[provider] = alias
		}
	}
}

// GetProviderAlias returns the primary short alias for a canonical provider ID, or providerID itself.
func GetProviderAlias(providerID string) string {
	if alias, ok := ProviderToAliasMap[providerID]; ok && alias != "" {
		return alias
	}
	return providerID
}
