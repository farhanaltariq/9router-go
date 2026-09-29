package chat

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	json "encoding/json/v2"

	"9router/proxy/internal/proxy/executor"
)

func TestIntegration_CommandCode_TwoTurnToolCalling(t *testing.T) {
	repo, cleanup := getRealUserDB(t)
	defer cleanup()

	conns, err := repo.GetProviderConnections("commandcode", true)
	if err != nil || len(conns) == 0 {
		t.Skip("no active commandcode connections")
	}

	executor.RegisterAll()
	handler := NewChatHandler(repo)

	// Turn 1: request tool call
	t1Body := `{
		"model": "commandcode",
		"stream": true,
		"messages": [
			{"role": "user", "content": "What is the weather in Tokyo right now?"}
		],
		"tools": [
			{
				"type": "function",
				"function": {
					"name": "get_current_weather",
					"description": "Get current weather for a city",
					"parameters": {
						"type": "object",
						"properties": {
							"location": {"type": "string", "description": "City name"}
						},
						"required": ["location"]
					}
				}
			}
		]
	}`

	req := httptest.NewRequest("POST", "/v1/chat/completions", bytes.NewReader([]byte(t1Body)))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	handler.HandleChatCompletions(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("Turn 1 expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var toolCallID, toolName, toolArgs, finishReason string
	for _, line := range strings.Split(rec.Body.String(), "\n") {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "data:") || strings.Contains(line, "[DONE]") {
			continue
		}
		data := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
		var chunk struct {
			Choices []struct {
				Delta struct {
					ToolCalls []struct {
						ID       string `json:"id"`
						Function struct {
							Name      string `json:"name"`
							Arguments string `json:"arguments"`
						} `json:"function"`
					} `json:"tool_calls"`
				} `json:"delta"`
				FinishReason string `json:"finish_reason"`
			} `json:"choices"`
		}
		if err := json.Unmarshal([]byte(data), &chunk); err == nil && len(chunk.Choices) > 0 {
			if len(chunk.Choices[0].Delta.ToolCalls) > 0 {
				tc := chunk.Choices[0].Delta.ToolCalls[0]
				if tc.ID != "" {
					toolCallID = tc.ID
				}
				if tc.Function.Name != "" {
					toolName = tc.Function.Name
				}
				toolArgs += tc.Function.Arguments
			}
			if chunk.Choices[0].FinishReason != "" {
				finishReason = chunk.Choices[0].FinishReason
			}
		}
	}

	if toolName != "get_current_weather" {
		t.Errorf("expected tool name get_current_weather, got %s", toolName)
	}
	if !strings.Contains(toolArgs, "Tokyo") {
		t.Errorf("expected toolArgs to contain Tokyo, got %s", toolArgs)
	}
	if finishReason != "tool_calls" {
		t.Errorf("expected finishReason tool_calls, got %s", finishReason)
	}

	// Turn 2: return tool output
	t2Req := map[string]any{
		"model":  "commandcode",
		"stream": true,
		"messages": []any{
			map[string]any{"role": "user", "content": "What is the weather in Tokyo right now?"},
			map[string]any{
				"role": "assistant",
				"tool_calls": []any{
					map[string]any{
						"id":   toolCallID,
						"type": "function",
						"function": map[string]any{
							"name":      toolName,
							"arguments": toolArgs,
						},
					},
				},
			},
			map[string]any{
				"role":         "tool",
				"tool_call_id": toolCallID,
				"content":      `{"temperature": 22, "condition": "sunny"}`,
			},
		},
		"tools": []any{
			map[string]any{
				"type": "function",
				"function": map[string]any{
					"name":        "get_current_weather",
					"description": "Get current weather for a city",
					"parameters": map[string]any{
						"type": "object",
						"properties": map[string]any{
							"location": map[string]any{"type": "string"},
						},
						"required": []string{"location"},
					},
				},
			},
		},
	}

	t2JSON, _ := json.Marshal(t2Req)
	req2 := httptest.NewRequest("POST", "/v1/chat/completions", bytes.NewReader(t2JSON))
	req2.Header.Set("Content-Type", "application/json")
	rec2 := httptest.NewRecorder()
	handler.HandleChatCompletions(rec2, req2)

	if rec2.Code != http.StatusOK {
		t.Fatalf("Turn 2 expected 200, got %d: %s", rec2.Code, rec2.Body.String())
	}

	var content, t2FinishReason string
	for _, line := range strings.Split(rec2.Body.String(), "\n") {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "data:") || strings.Contains(line, "[DONE]") {
			continue
		}
		data := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
		var chunk struct {
			Choices []struct {
				Delta struct {
					Content string `json:"content"`
				} `json:"delta"`
				FinishReason string `json:"finish_reason"`
			} `json:"choices"`
		}
		if err := json.Unmarshal([]byte(data), &chunk); err == nil && len(chunk.Choices) > 0 {
			content += chunk.Choices[0].Delta.Content
			if chunk.Choices[0].FinishReason != "" {
				t2FinishReason = chunk.Choices[0].FinishReason
			}
		}
	}

	if content == "" {
		t.Fatalf("expected non-empty message content in Turn 2, got: %s", rec2.Body.String())
	}
	if t2FinishReason != "stop" {
		t.Errorf("expected finish_reason stop in Turn 2, got %s", t2FinishReason)
	}
}

func TestIntegration_Antigravity_TwoTurnToolCalling(t *testing.T) {
	repo, cleanup := getRealUserDB(t)
	defer cleanup()

	conns, err := repo.GetProviderConnections("antigravity", true)
	if err != nil || len(conns) == 0 {
		t.Skip("no active antigravity connections")
	}

	executor.RegisterAll()
	handler := NewChatHandler(repo)

	// Turn 1: request tool call
	t1Body := `{
		"model": "antigravity",
		"stream": false,
		"messages": [
			{"role": "user", "content": "What is the weather in Tokyo right now?"}
		],
		"tools": [
			{
				"type": "function",
				"function": {
					"name": "get_current_weather",
					"description": "Get current weather for a city",
					"parameters": {
						"type": "object",
						"properties": {
							"location": {"type": "string", "description": "City name"}
						},
						"required": ["location"]
					}
				}
			}
		]
	}`

	req := httptest.NewRequest("POST", "/v1/chat/completions", bytes.NewReader([]byte(t1Body)))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	handler.HandleChatCompletions(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("Turn 1 expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var t1Resp struct {
		Choices []struct {
			Message struct {
				Role      string `json:"role"`
				ToolCalls []struct {
					ID       string `json:"id"`
					Type     string `json:"type"`
					Function struct {
						Name      string `json:"name"`
						Arguments string `json:"arguments"`
					} `json:"function"`
				} `json:"tool_calls"`
			} `json:"message"`
			FinishReason string `json:"finish_reason"`
		} `json:"choices"`
	}

	if err := json.Unmarshal(rec.Body.Bytes(), &t1Resp); err != nil {
		t.Fatalf("unmarshal Turn 1 response: %v", err)
	}
	if len(t1Resp.Choices) == 0 || len(t1Resp.Choices[0].Message.ToolCalls) == 0 {
		t.Fatalf("expected tool_calls in Turn 1, got body: %s", rec.Body.String())
	}

	tc := t1Resp.Choices[0].Message.ToolCalls[0]
	if tc.Function.Name != "get_current_weather" {
		t.Errorf("expected tool name get_current_weather, got %s", tc.Function.Name)
	}
	if !strings.Contains(tc.Function.Arguments, "Tokyo") {
		t.Errorf("expected argument location to contain Tokyo, got %s", tc.Function.Arguments)
	}
	if t1Resp.Choices[0].FinishReason != "tool_calls" {
		t.Errorf("expected finish_reason tool_calls, got %s", t1Resp.Choices[0].FinishReason)
	}

	// Turn 2: return tool output
	t2Req := map[string]any{
		"model":  "antigravity",
		"stream": false,
		"messages": []any{
			map[string]any{"role": "user", "content": "What is the weather in Tokyo right now?"},
			map[string]any{
				"role": "assistant",
				"tool_calls": []any{
					map[string]any{
						"id":   tc.ID,
						"type": "function",
						"function": map[string]any{
							"name":      tc.Function.Name,
							"arguments": tc.Function.Arguments,
						},
					},
				},
			},
			map[string]any{
				"role":         "tool",
				"tool_call_id": tc.ID,
				"content":      `{"temperature": 22, "condition": "sunny"}`,
			},
		},
		"tools": []any{
			map[string]any{
				"type": "function",
				"function": map[string]any{
					"name":        "get_current_weather",
					"description": "Get current weather for a city",
					"parameters": map[string]any{
						"type": "object",
						"properties": map[string]any{
							"location": map[string]any{"type": "string"},
						},
						"required": []string{"location"},
					},
				},
			},
		},
	}

	t2JSON, _ := json.Marshal(t2Req)
	req2 := httptest.NewRequest("POST", "/v1/chat/completions", bytes.NewReader(t2JSON))
	req2.Header.Set("Content-Type", "application/json")
	rec2 := httptest.NewRecorder()
	handler.HandleChatCompletions(rec2, req2)

	if rec2.Code != http.StatusOK {
		t.Fatalf("Turn 2 expected 200, got %d: %s", rec2.Code, rec2.Body.String())
	}

	var t2Resp struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
			FinishReason string `json:"finish_reason"`
		} `json:"choices"`
	}
	if err := json.Unmarshal(rec2.Body.Bytes(), &t2Resp); err != nil {
		t.Fatalf("unmarshal Turn 2 response: %v", err)
	}
	if len(t2Resp.Choices) == 0 || t2Resp.Choices[0].Message.Content == "" {
		t.Fatalf("expected non-empty message content in Turn 2, got: %s", rec2.Body.String())
	}
	if t2Resp.Choices[0].FinishReason != "stop" {
		t.Errorf("expected finish_reason stop in Turn 2, got %s", t2Resp.Choices[0].FinishReason)
	}
}
