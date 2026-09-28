package executor

import (
	json "encoding/json/v2"
	"testing"
)

func TestBuildCommandcodeBody(t *testing.T) {
	const in = `{
		"model": "m",
		"messages": [
			{"role": "system", "content": "be brief"},
			{"role": "user", "content": "hi"},
			{"role": "assistant", "content": "calling", "tool_calls": [
				{"id": "tc1", "type": "function", "function": {"name": "get", "arguments": "{\"url\":\"x\"}"}}
			]},
			{"role": "tool", "tool_call_id": "tc1", "name": "get", "content": "ok"}
		],
		"tools": [
			{"type": "function", "function": {"name": "get", "description": "d", "parameters": {"type": "object", "properties": {"url": {"type": "string"}}}}}
		]
	}`
	got, err := buildCommandcodeBody([]byte(in), "m")
	if err != nil {
		t.Fatalf("buildCommandcodeBody: %v", err)
	}

	var payload struct {
		Params struct {
			Messages []struct {
				Role    string `json:"role"`
				Content []struct {
					Type       string `json:"type"`
					Text       string `json:"text"`
					ToolCallID string `json:"toolCallId"`
					ToolName   string `json:"toolName"`
					Output     struct {
						Type  string `json:"type"`
						Value string `json:"value"`
					} `json:"output"`
					Input map[string]any `json:"input"`
				} `json:"content"`
			} `json:"messages"`
			System string `json:"system"`
			Tools  []struct {
				Name        string         `json:"name"`
				Description string         `json:"description"`
				Schema      map[string]any `json:"input_schema"`
			} `json:"tools"`
		} `json:"params"`
		ThreadID string `json:"threadId"`
	}
	if err := json.Unmarshal(got, &payload); err != nil {
		t.Fatalf("unmarshal payload: %v", err)
	}

	if payload.ThreadID == "" {
		t.Error("missing threadId")
	}
	if payload.Params.System != "be brief" {
		t.Errorf("system = %q, want %q", payload.Params.System, "be brief")
	}
	if len(payload.Params.Messages) != 3 {
		t.Fatalf("messages = %d, want 3 (system hoisted)", len(payload.Params.Messages))
	}

	// assistant with tool_calls → reasoning + text + tool-call
	assistant := payload.Params.Messages[1]
	if assistant.Role != "assistant" {
		t.Fatalf("first msg role = %q, want assistant", assistant.Role)
	}
	types := make([]string, 0, len(assistant.Content))
	for _, b := range assistant.Content {
		types = append(types, b.Type)
	}
	wantTypes := []string{"reasoning", "text", "tool-call"}
	if len(types) != len(wantTypes) {
		t.Fatalf("assistant blocks = %v, want %v", types, wantTypes)
	}
	for i := range wantTypes {
		if types[i] != wantTypes[i] {
			t.Fatalf("assistant blocks = %v, want %v", types, wantTypes)
		}
	}
	tc := assistant.Content[2]
	if tc.ToolCallID != "tc1" || tc.ToolName != "get" || tc.Input["url"] != "x" {
		t.Errorf("tool-call block = %+v", tc)
	}

	// tool → tool-result with toolCallId/toolName/output
	tool := payload.Params.Messages[2]
	if tool.Role != "tool" {
		t.Fatalf("second msg role = %q, want tool", tool.Role)
	}
	tr := tool.Content[0]
	if tr.Type != "tool-result" || tr.ToolCallID != "tc1" || tr.ToolName != "get" {
		t.Errorf("tool-result block = %+v", tr)
	}
	if tr.Output.Type != "text" || tr.Output.Value != "ok" {
		t.Errorf("tool-result output = %+v", tr.Output)
	}

	// tools → Anthropic plain shape
	if len(payload.Params.Tools) != 1 {
		t.Fatalf("tools = %d, want 1", len(payload.Params.Tools))
	}
	if payload.Params.Tools[0].Name != "get" || payload.Params.Tools[0].Description != "d" || payload.Params.Tools[0].Schema["type"] != "object" {
		t.Errorf("tool[0] = %+v", payload.Params.Tools[0])
	}
}

func TestBuildCommandcodeBody_StripsInvalidReasoningEffort(t *testing.T) {
	for _, val := range []string{"very_high", "auto", "turbo", ""} {
		in := `{"model":"m","reasoning_effort":"` + val + `","messages":[{"role":"user","content":"hi"}]}`
		got, err := buildCommandcodeBody([]byte(in), "m")
		if err != nil {
			t.Fatalf("buildCommandcodeBody: %v", err)
		}
		var p struct{ Params map[string]any `json:"params"` }
		if err := json.Unmarshal(got, &p); err != nil {
			t.Fatal(err)
		}
		if _, ok := p.Params["reasoning_effort"]; ok {
			t.Errorf("reasoning_effort=%q should be stripped, but was kept", val)
		}
	}
	// valid values must be preserved
	for _, val := range []string{"low", "medium", "high", "xhigh", "max"} {
		in := `{"model":"m","reasoning_effort":"` + val + `","messages":[{"role":"user","content":"hi"}]}`
		got, err := buildCommandcodeBody([]byte(in), "m")
		if err != nil {
			t.Fatalf("buildCommandcodeBody: %v", err)
		}
		var p struct{ Params map[string]any `json:"params"` }
		if err := json.Unmarshal(got, &p); err != nil {
			t.Fatal(err)
		}
		if p.Params["reasoning_effort"] != val {
			t.Errorf("reasoning_effort=%q should be preserved, got %v", val, p.Params["reasoning_effort"])
		}
	}
}
