package oauth

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	json "encoding/json/v2"
	"strings"
)

// connectionDisplayName is the single naming rule for every OAuth-family
// connection: the account email when known, else an explicit user-supplied
// name, else the provider default. Naming accounts "Provider (name)" makes
// multi-account setups indistinguishable on the provider detail page, so the
// provider prefix is intentionally dropped here.
func connectionDisplayName(provider, name, email, fallback string) string {
	if email != "" {
		return email
	}
	if strings.TrimSpace(name) != "" {
		return name
	}
	if fallback != "" {
		return fallback
	}
	return provider
}

func shortHash(s string) string {
	sum := sha256.Sum256([]byte(s))
	return hex.EncodeToString(sum[:])[:8]
}

func capitalize(s string) string {
	if s == "" {
		return ""
	}
	return strings.ToUpper(s[:1]) + s[1:]
}

func extractEmailFromJWT(token string) string {
	parts := strings.Split(token, ".")
	if len(parts) < 2 {
		return ""
	}
	payload := parts[1]
	if l := len(payload) % 4; l > 0 {
		payload += strings.Repeat("=", 4-l)
	}
	data, err := base64.URLEncoding.DecodeString(payload)
	if err != nil {
		data, err = base64.RawURLEncoding.DecodeString(parts[1])
		if err != nil {
			return ""
		}
	}
	var claims map[string]any
	if err := json.Unmarshal(data, &claims); err != nil {
		return ""
	}
	for _, k := range []string{"email", "unique_name", "preferred_username", "sub"} {
		if v, ok := claims[k].(string); ok && v != "" && strings.Contains(v, "@") {
			return v
		}
	}
	return ""
}
