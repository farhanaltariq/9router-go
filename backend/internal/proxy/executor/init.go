package executor

// RegisterAll registers all known executor factories.
// Called once at init or server startup.
func RegisterAll() {
	Register("openai", func() Executor { return ForwardOpenAI })
	Register("anthropic", func() Executor { return ForwardOpenAI })
	Register("nvidia", func() Executor { return ForwardOpenAI })
	Register("github", func() Executor { return ForwardOpenAI })
	Register("ollama", func() Executor { return ForwardOpenAI })
	Register("opencode", func() Executor { return ForwardOpenAI })
	Register("cloudflare-ai", func() Executor { return ForwardOpenAI })
	Register("codebuddy-cn", func() Executor { return ForwardCodebuddyCN })
	Register("codebuddy-intl", func() Executor { return ForwardCodebuddyCN })
	Register("codex", func() Executor { return ForwardCodex })
	Register("commandcode", func() Executor { return ForwardCommandcode })
}
