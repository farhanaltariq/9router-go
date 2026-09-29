package chat

import (
	"errors"
	"io"
	json "encoding/json/v2"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"9router/proxy/internal/providers"
	"9router/proxy/internal/proxy/executor"
)

func TestProcessCommandcodeEvent_TextDelta(t *testing.T) {
	state := &executor.CommandcodeStreamState{
		ResponseID: "test-id",
		Created:    1000,
	}
	event := map[string]any{"type": "text-delta", "text": "Hello world"}
	chunks := executor.ProcessCommandcodeEvent(event, "text-delta", state)
	if len(chunks) == 0 {
		t.Fatal("expected output chunks")
	}
	if !strings.Contains(chunks[0], "Hello world") {
		t.Errorf("expected content in chunk, got %s", chunks[0])
	}
	if state.OutputLength != 11 {
		t.Errorf("expected outputLength 11, got %d", state.OutputLength)
	}
	if state.ChunkIndex != 1 {
		t.Errorf("expected chunkIndex 1, got %d", state.ChunkIndex)
	}
}

func TestProcessCommandcodeEvent_ReasoningDelta(t *testing.T) {
	state := &executor.CommandcodeStreamState{ResponseID: "test-id", Created: 1000}
	event := map[string]any{"type": "reasoning-delta", "text": "thinking step by step"}
	chunks := executor.ProcessCommandcodeEvent(event, "reasoning-delta", state)
	if len(chunks) == 0 {
		t.Fatal("expected output chunks")
	}
	if !strings.Contains(chunks[0], "reasoning_content") {
		t.Errorf("expected reasoning_content, got %s", chunks[0])
	}
}

func TestProcessCommandcodeEvent_ToolInputStart(t *testing.T) {
	state := &executor.CommandcodeStreamState{ResponseID: "test-id", Created: 1000}
	event := map[string]any{
		"type":     "tool-input-start",
		"id":       "call_123",
		"toolName": "get_weather",
	}
	chunks := executor.ProcessCommandcodeEvent(event, "tool-input-start", state)
	if len(chunks) == 0 {
		t.Fatal("expected output chunks")
	}
	if !strings.Contains(chunks[0], "get_weather") {
		t.Errorf("expected tool name, got %s", chunks[0])
	}
	if state.ToolIndex != 1 {
		t.Errorf("expected toolIndex 1, got %d", state.ToolIndex)
	}
}

func TestProcessCommandcodeEvent_ToolInputDelta(t *testing.T) {
	state := &executor.CommandcodeStreamState{ResponseID: "test-id", Created: 1000}
	state.ToolIndexByID = map[string]int{"call_123": 0}
	event := map[string]any{
		"type":  "tool-input-delta",
		"id":    "call_123",
		"delta": `{"location":"Jakarta"}`,
	}
	chunks := executor.ProcessCommandcodeEvent(event, "tool-input-delta", state)
	if len(chunks) == 0 {
		t.Fatal("expected output chunks")
	}
	if !strings.Contains(chunks[0], "Jakarta") {
		t.Errorf("expected arguments, got %s", chunks[0])
	}
}

func TestProcessCommandcodeEvent_ToolCall(t *testing.T) {
	state := &executor.CommandcodeStreamState{ResponseID: "test-id", Created: 1000}
	event := map[string]any{
		"type":       "tool-call",
		"toolCallId": "call_456",
		"toolName":   "search",
		"input":      map[string]any{"query": "test"},
	}
	chunks := executor.ProcessCommandcodeEvent(event, "tool-call", state)
	if len(chunks) == 0 {
		t.Fatal("expected output chunks")
	}
	if !strings.Contains(chunks[0], "search") {
		t.Errorf("expected function name, got %s", chunks[0])
	}
	if !strings.Contains(chunks[0], "test") {
		t.Errorf("expected input in arguments, got %s", chunks[0])
	}
}

func TestProcessCommandcodeEvent_FinishStep(t *testing.T) {
	state := &executor.CommandcodeStreamState{ResponseID: "test-id", Created: 1000}
	event := map[string]any{
		"type":          "finish-step",
		"finishReason": "stop",
	}
	chunks := executor.ProcessCommandcodeEvent(event, "finish-step", state)
	if len(chunks) != 0 {
		t.Errorf("expected no chunks from finish-step, got %d", len(chunks))
	}
	if state.FinishReason != "stop" {
		t.Errorf("expected finishReason 'stop', got %q", state.FinishReason)
	}
}

func TestProcessCommandcodeEvent_Finish(t *testing.T) {
	state := &executor.CommandcodeStreamState{
		ResponseID: "test-id",
		Created:    1000,
	}
	event := map[string]any{"type": "finish", "finishReason": "stop"}
	chunks := executor.ProcessCommandcodeEvent(event, "finish", state)
	if len(chunks) == 0 {
		t.Fatal("expected output from finish")
	}
	if !strings.Contains(chunks[0], `"finish_reason":"stop"`) {
		t.Errorf("expected finish_reason, got %s", chunks[0])
	}
	if !state.Finished {
		t.Error("expected state.Finished=true")
	}
}

func TestProcessCommandcodeEvent_Error(t *testing.T) {
	state := &executor.CommandcodeStreamState{ResponseID: "test-id", Created: 1000}
	event := map[string]any{
		"type":  "error",
		"error": "rate limit exceeded",
	}
	chunks := executor.ProcessCommandcodeEvent(event, "error", state)
	if len(chunks) == 0 {
		t.Fatal("expected error output chunks")
	}
	combined := strings.Join(chunks, "")
	if !strings.Contains(combined, "rate limit exceeded") {
		t.Errorf("expected error message, got %s", combined)
	}
	if !state.Finished {
		t.Error("expected state.Finished=true after error")
	}
}

func TestBuildCommandcodeChunk(t *testing.T) {
	state := &executor.CommandcodeStreamState{ResponseID: "test-id", Created: 1000, Model: "deepseek-v4"}
	result := executor.BuildCommandcodeChunk(state, map[string]any{"content": "hi"}, "stop")
	if !strings.Contains(result, "deepseek-v4") {
		t.Errorf("expected model in chunk, got %s", result)
	}
	if !strings.Contains(result, `"finish_reason":"stop"`) {
		t.Errorf("expected finish_reason, got %s", result)
	}
}

func TestForwardCommandcodeRequest_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("x-command-code-version") != "0.25.7" {
			t.Errorf("expected x-command-code-version header, got %q", r.Header.Get("x-command-code-version"))
		}
		if r.Header.Get("x-cli-environment") != "cli" {
			t.Errorf("expected x-cli-environment header")
		}
		if r.Header.Get("x-session-id") == "" {
			t.Errorf("expected x-session-id header")
		}
		if r.Header.Get("User-Agent") != "commandcode/0.25.7 (cli)" {
			t.Errorf("expected User-Agent header 'commandcode/0.25.7 (cli)', got %q", r.Header.Get("User-Agent"))
		}
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"type":"text-delta","text":"commandcode response"}` + "\n" +
			`{"type":"finish","finishReason":"stop"}` + "\n"))
	}))
	defer srv.Close()

	cfg := &providers.ProviderConfig{
		BaseURL: srv.URL,
	}
	body := []byte(`{"model":"deepseek-v4","messages":[{"role":"user","content":"hi"}]}`)
	rec := httptest.NewRecorder()
	err := executor.ForwardCommandcode(rec, &executor.Request{
		Client:   srv.Client(),
		Config:   cfg,
		APIKey:   "sk-cc",
		Body:     body,
		IsStream: true,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(rec.Body.String(), "commandcode response") {
		t.Errorf("expected response content, got %s", rec.Body.String())
	}
}

func TestForwardCommandcodeRequest_UpstreamError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"error":"unauthorized"}`))
	}))
	defer srv.Close()

	cfg := &providers.ProviderConfig{
		BaseURL: srv.URL,
	}
	body := []byte(`{"model":"x","messages":[]}`)
	rec := httptest.NewRecorder()
	err := executor.ForwardCommandcode(rec, &executor.Request{
		Client:   srv.Client(),
		Config:   cfg,
		APIKey:   "bad-key",
		Body:     body,
		IsStream: true,
	})
	if err == nil {
		t.Fatal("expected error for 401")
	}
	var ue *upstreamError
	if !errors.As(err, &ue) {
		t.Fatalf("expected *upstreamError, got %T", err)
	}
	if ue.StatusCode != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", ue.StatusCode)
	}
}

func TestForwardCommandcodeRequest_StaticHeaders(t *testing.T) {
	var capturedUA string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedUA = r.Header.Get("User-Agent")
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"type":"finish","finishReason":"stop"}` + "\n"))
	}))
	defer srv.Close()

	cfg := &providers.ProviderConfig{
		BaseURL: srv.URL,
		StaticHeaders: map[string]string{
			"User-Agent": "custom-cc-agent/1.0",
		},
	}
	body := []byte(`{"model":"deepseek-v4","messages":[{"role":"user","content":"hi"}]}`)
	rec := httptest.NewRecorder()
	err := executor.ForwardCommandcode(rec, &executor.Request{
		Client:   srv.Client(),
		Config:   cfg,
		APIKey:   "sk-cc",
		Body:     body,
		IsStream: true,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if capturedUA != "custom-cc-agent/1.0" {
		t.Errorf("expected User-Agent 'custom-cc-agent/1.0', got %q", capturedUA)
	}
}

func TestForwardCommandcodeRequest_RelayHeaders(t *testing.T) {
	var gotTarget, gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotTarget = r.Header.Get("x-relay-target")
		gotPath = r.Header.Get("x-relay-path")
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"type":"finish","finishReason":"stop"}` + "\n"))
	}))
	defer srv.Close()

	cfg := &providers.ProviderConfig{
		BaseURL: srv.URL,
		StaticHeaders: map[string]string{
			"x-relay-target": "https://api.commandcode.ai",
			"x-relay-path":   "/alpha/chat",
		},
	}
	body := []byte(`{"model":"deepseek-v4","messages":[{"role":"user","content":"hi"}]}`)
	rec := httptest.NewRecorder()
	err := executor.ForwardCommandcode(rec, &executor.Request{
		Client:   srv.Client(),
		Config:   cfg,
		APIKey:   "sk-cc",
		Body:     body,
		IsStream: true,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotTarget != "https://api.commandcode.ai" {
		t.Errorf("expected x-relay-target https://api.commandcode.ai, got %q", gotTarget)
	}
	if gotPath != "/alpha/chat" {
		t.Errorf("expected x-relay-path /alpha/chat, got %q", gotPath)
	}
}

func TestForwardCommandcodeRequest_ImageAndReasoningEffort(t *testing.T) {
	var capturedBody map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		data, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(data, &capturedBody)
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"type":"finish","finishReason":"stop"}` + "\n"))
	}))
	defer srv.Close()

	cfg := &providers.ProviderConfig{
		BaseURL: srv.URL,
	}
	body := []byte(`{
		"model": "deepseek-v4-vision",
		"reasoning_effort": "high",
		"messages": [
			{
				"role": "user",
				"content": [
					{"type": "text", "text": "what is this?"},
					{"type": "image_url", "image_url": {"url": "data:image/png;base64,iVBORw0KGgo="}}
				]
			}
		]
	}`)
	rec := httptest.NewRecorder()
	err := executor.ForwardCommandcode(rec, &executor.Request{
		Client:   srv.Client(),
		Config:   cfg,
		APIKey:   "sk-test",
		Body:     body,
		IsStream: true,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	params, _ := capturedBody["params"].(map[string]any)
	if params == nil {
		t.Fatal("expected params object in payload")
	}
	if effort, _ := params["reasoning_effort"].(string); effort != "high" {
		t.Errorf("expected reasoning_effort 'high', got %v", params["reasoning_effort"])
	}

	msgs, _ := params["messages"].([]any)
	if len(msgs) != 1 {
		t.Fatalf("expected 1 message, got %d", len(msgs))
	}
	userMsg, _ := msgs[0].(map[string]any)
	content, _ := userMsg["content"].([]any)
	if len(content) != 2 {
		t.Fatalf("expected 2 content blocks, got %d", len(content))
	}
	imgPart, _ := content[1].(map[string]any)
	if imgPart["type"] != "image" {
		t.Errorf("expected content[1].type 'image', got %v", imgPart["type"])
	}
	if imgPart["image"] != "data:image/png;base64,iVBORw0KGgo=" {
		t.Errorf("expected data URI preserved, got %v", imgPart["image"])
	}
	if imgPart["mimeType"] != "image/png" {
		t.Errorf("expected mimeType 'image/png', got %v", imgPart["mimeType"])
	}
}

func TestForwardCommandcode_ToolCallFinishReason(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		// Simulate commandcode sending tool-input-start + tool-call + finish
		events := []string{
			`{"type":"text-delta","text":"Let me check"}`,
			`{"type":"tool-input-start","id":"call_abc","toolName":"get_weather"}`,
			`{"type":"tool-input-delta","id":"call_abc","delta":"{\"city\":\"Jakarta\"}"}`,
			`{"type":"tool-call","toolCallId":"call_abc","toolName":"get_weather","input":{"city":"Jakarta"}}`,
			`{"type":"finish","finishReason":"stop"}`,
		}
		for _, ev := range events {
			w.Write([]byte(ev + "\n"))
		}
	}))
	defer srv.Close()

	cfg := &providers.ProviderConfig{BaseURL: srv.URL}
	body := []byte(`{"model":"deepseek-v4","messages":[{"role":"user","content":"Weather in Jakarta"}]}`)
	rec := httptest.NewRecorder()
	err := executor.ForwardCommandcode(rec, &executor.Request{
		Client:   srv.Client(),
		Config:   cfg,
		APIKey:   "sk-cc",
		Body:     body,
		IsStream: true,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	respBody := rec.Body.String()
	t.Logf("response body: %s", respBody)

	if !strings.Contains(respBody, "get_weather") {
		t.Errorf("expected get_weather tool name in response, got: %s", respBody)
	}
	if !strings.Contains(respBody, "Jakarta") {
		t.Errorf("expected Jakarta args in response, got: %s", respBody)
	}
	if !strings.Contains(respBody, `"finish_reason":"tool_calls"`) && !strings.Contains(respBody, `"finish_reason": "tool_calls"`) {
		t.Errorf("expected finish_reason tool_calls in response, got: %s", respBody)
	}
	if strings.Contains(respBody, `"finish_reason":"stop"`) || strings.Contains(respBody, `"finish_reason": "stop"`) {
		t.Errorf("finish_reason must NOT be stop when tool calls present, got: %s", respBody)
	}
}

func TestForwardCommandcode_TextOnlyFinishReasonStop(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		events := []string{
			`{"type":"text-delta","text":"Hello world"}`,
			`{"type":"finish","finishReason":"stop"}`,
		}
		for _, ev := range events {
			w.Write([]byte(ev + "\n"))
		}
	}))
	defer srv.Close()

	cfg := &providers.ProviderConfig{BaseURL: srv.URL}
	body := []byte(`{"model":"deepseek-v4","messages":[{"role":"user","content":"hi"}]}`)
	rec := httptest.NewRecorder()
	err := executor.ForwardCommandcode(rec, &executor.Request{
		Client:   srv.Client(),
		Config:   cfg,
		APIKey:   "sk-cc",
		Body:     body,
		IsStream: true,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	respBody := rec.Body.String()
	if !strings.Contains(respBody, "Hello world") {
		t.Errorf("expected text in response, got: %s", respBody)
	}
	if strings.Contains(respBody, `"finish_reason":"tool_calls"`) || strings.Contains(respBody, `"finish_reason": "tool_calls"`) {
		t.Errorf("text-only stream must not get tool_calls finish_reason, got: %s", respBody)
	}
}

// TestForwardCommandcode_MultiTurnToolExecution simulates a complete real-world multi-turn tool flow:
// Turn 1: User asks question -> model streams tool call -> client parses tool call & finish_reason="tool_calls".
// Turn 2: Client sends tool execution result (role: "tool" without name) -> CommandCode upstream receives
//
//	properly formatted tool-result with mapped toolName -> model streams final answer.
func TestForwardCommandcode_MultiTurnToolExecution(t *testing.T) {
	turn := 1
	var capturedTurn2Body map[string]any

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)

		if turn == 1 {
			// Turn 1: model calls tool get_weather
			events := []string{
				`{"type":"tool-call","toolCallId":"call_xyz789","toolName":"get_weather","input":{"location":"Tokyo"}}`,
				`{"type":"finish-step","finishReason":"tool-calls"}`,
				`{"type":"finish"}`,
			}
			for _, ev := range events {
				w.Write([]byte(ev + "\n"))
			}
		} else {
			// Turn 2: capture request body sent to upstream, then reply with text
			data, _ := io.ReadAll(r.Body)
			_ = json.Unmarshal(data, &capturedTurn2Body)

			events := []string{
				`{"type":"text-delta","text":"The weather in Tokyo is 22°C."}`,
				`{"type":"finish-step","finishReason":"stop"}`,
				`{"type":"finish"}`,
			}
			for _, ev := range events {
				w.Write([]byte(ev + "\n"))
			}
		}
	}))
	defer srv.Close()

	cfg := &providers.ProviderConfig{BaseURL: srv.URL}

	// --- Turn 1 ---
	turn1Body := []byte(`{
		"model": "deepseek-v4",
		"messages": [{"role": "user", "content": "Weather in Tokyo?"}],
		"tools": [{
			"type": "function",
			"function": {"name": "get_weather", "parameters": {"type": "object", "properties": {"location": {"type": "string"}}}}
		}]
	}`)
	rec1 := httptest.NewRecorder()
	err := executor.ForwardCommandcode(rec1, &executor.Request{
		Client:   srv.Client(),
		Config:   cfg,
		APIKey:   "sk-cc",
		Body:     turn1Body,
		IsStream: true,
	})
	if err != nil {
		t.Fatalf("turn 1 error: %v", err)
	}

	body1 := rec1.Body.String()
	t.Logf("Turn 1 stream response:\n%s", body1)

	// Verify index is present in tool-call
	if !strings.Contains(body1, `"index":0`) && !strings.Contains(body1, `"index": 0`) {
		t.Errorf("expected 'index': 0 in tool-call chunk, got: %s", body1)
	}
	// Verify tool_calls finish_reason
	if !strings.Contains(body1, `"finish_reason":"tool_calls"`) && !strings.Contains(body1, `"finish_reason": "tool_calls"`) {
		t.Errorf("expected finish_reason 'tool_calls' in turn 1, got: %s", body1)
	}

	// --- Turn 2 ---
	turn = 2
	// Client sends tool execution result without "name" field (standard OpenAI SDK format)
	turn2Body := []byte(`{
		"model": "deepseek-v4",
		"messages": [
			{"role": "user", "content": "Weather in Tokyo?"},
			{"role": "assistant", "tool_calls": [{"id": "call_xyz789", "type": "function", "function": {"name": "get_weather", "arguments": "{\"location\":\"Tokyo\"}"}}]},
			{"role": "tool", "tool_call_id": "call_xyz789", "content": "{\"temperature\": 22, \"condition\": \"clear\"}"}
		]
	}`)
	rec2 := httptest.NewRecorder()
	err = executor.ForwardCommandcode(rec2, &executor.Request{
		Client:   srv.Client(),
		Config:   cfg,
		APIKey:   "sk-cc",
		Body:     turn2Body,
		IsStream: true,
	})
	if err != nil {
		t.Fatalf("turn 2 error: %v", err)
	}

	body2 := rec2.Body.String()
	t.Logf("Turn 2 stream response:\n%s", body2)

	if !strings.Contains(body2, "The weather in Tokyo is 22°C.") {
		t.Errorf("expected final answer in turn 2, got: %s", body2)
	}
	if !strings.Contains(body2, `"finish_reason":"stop"`) && !strings.Contains(body2, `"finish_reason": "stop"`) {
		t.Errorf("expected finish_reason 'stop' in turn 2, got: %s", body2)
	}

	// Verify upstream CommandCode received correctly mapped toolName
	params, _ := capturedTurn2Body["params"].(map[string]any)
	msgs, _ := params["messages"].([]any)
	if len(msgs) != 3 {
		t.Fatalf("expected 3 messages in upstream payload, got %d", len(msgs))
	}
	toolMsg, _ := msgs[2].(map[string]any)
	if toolMsg["role"] != "tool" {
		t.Errorf("expected role 'tool', got %v", toolMsg["role"])
	}
	contentBlocks, _ := toolMsg["content"].([]any)
	if len(contentBlocks) == 0 {
		t.Fatalf("expected content blocks in tool message")
	}
	toolResultBlock, _ := contentBlocks[0].(map[string]any)
	if toolResultBlock["toolName"] != "get_weather" {
		t.Errorf("expected toolName 'get_weather' mapped from assistant tool_call, got %q", toolResultBlock["toolName"])
	}
	if toolResultBlock["toolCallId"] != "call_xyz789" {
		t.Errorf("expected toolCallId 'call_xyz789', got %q", toolResultBlock["toolCallId"])
	}
}
