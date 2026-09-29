package translator

import (
	"bytes"
	"strings"
	"testing"
)

func TestTranslateGeminiChunkToOpenAI_FunctionCallFinishReason(t *testing.T) {
	// Simulate a Gemini stream: functionCall chunk, then STOP finish chunk.
	fnChunk := `{
		"candidates": [{
			"content": {
				"parts": [{"functionCall": {"name": "get_weather", "args": {"city": "Jakarta"}}}]
			}
		}]
	}`

	finishChunk := `{
		"candidates": [{
			"content": {"parts": []},
			"finishReason": "STOP"
		}],
		"usageMetadata": {
			"promptTokenCount": 100,
			"candidatesTokenCount": 20
		}
	}`

	state := &GeminiStreamState{}

	// First chunk: function call
	out1, err := TranslateGeminiChunkToOpenAI([]byte(fnChunk), state)
	if err != nil {
		t.Fatalf("fn chunk: %v", err)
	}
	if len(out1) == 0 {
		t.Fatal("expected output from function call chunk")
	}
	if !state.HasToolCalls {
		t.Fatal("expected HasToolCalls=true after functionCall part")
	}

	// Second chunk: finish with STOP
	out2, err := TranslateGeminiChunkToOpenAI([]byte(finishChunk), state)
	if err != nil {
		t.Fatalf("finish chunk: %v", err)
	}
	if len(out2) == 0 {
		t.Fatal("expected output from finish chunk")
	}

	// Extract the last SSE data line from the output
	lastLine := lastSSEData(out2)
	if lastLine == "" {
		t.Fatal("no SSE data line found in finish output")
	}
	if !strings.Contains(lastLine, `"finish_reason":"tool_calls"`) &&
		!strings.Contains(lastLine, `"finish_reason": "tool_calls"`) {
		t.Errorf("expected finish_reason tool_calls in final chunk, got: %s", lastLine)
	}
}

func TestTranslateGeminiChunkToOpenAI_NoToolCallFinishReasonStop(t *testing.T) {
	// Plain text + STOP should remain "stop"
	textChunk := `{
		"candidates": [{
			"content": {"parts": [{"text": "Hello"}]},
			"finishReason": "STOP"
		}],
		"usageMetadata": {"promptTokenCount": 10, "candidatesTokenCount": 5}
	}`

	state := &GeminiStreamState{}
	out, err := TranslateGeminiChunkToOpenAI([]byte(textChunk), state)
	if err != nil {
		t.Fatalf("%v", err)
	}
	if state.HasToolCalls {
		t.Error("expected HasToolCalls=false for text-only chunk")
	}

	lastLine := lastSSEData(out)
	if strings.Contains(lastLine, `"finish_reason":"tool_calls"`) ||
		strings.Contains(lastLine, `"finish_reason": "tool_calls"`) {
		t.Errorf("text-only stream must not get tool_calls finish_reason: %s", lastLine)
	}
}

// lastSSEData extracts the JSON payload from the last "data: ..." line.
func lastSSEData(b []byte) string {
	var last string
	for _, line := range bytes.Split(b, []byte("\n")) {
		s := strings.TrimSpace(string(line))
		if strings.HasPrefix(s, "data: ") {
			last = strings.TrimPrefix(s, "data: ")
		}
	}
	return last
}

func TestTranslateGeminiChunkToOpenAI_CachedTokens(t *testing.T) {
	chunkJSON := `{
		"candidates": [
			{
				"content": {
					"parts": [{"text": "Hello world"}]
				},
				"finishReason": "STOP"
			}
		],
		"usageMetadata": {
			"promptTokenCount": 1000,
			"candidatesTokenCount": 50,
			"cachedContentTokenCount": 800
		}
	}`

	state := &GeminiStreamState{
		MessageId: "test-msg-1",
		Model:     "gemini-3.7-flash-high",
	}

	chunks, err := TranslateGeminiChunkToOpenAI([]byte(chunkJSON), state)
	if err != nil {
		t.Fatalf("TranslateGeminiChunkToOpenAI failed: %v", err)
	}

	if len(chunks) == 0 {
		t.Fatal("expected chunks, got 0")
	}

	if state.Usage == nil {
		t.Fatal("expected state.Usage to be non-nil")
	}

	if state.Usage.PromptTokens != 1000 {
		t.Errorf("expected 1000 prompt tokens, got %d", state.Usage.PromptTokens)
	}
	if state.Usage.CompletionTokens != 50 {
		t.Errorf("expected 50 completion tokens, got %d", state.Usage.CompletionTokens)
	}
	if state.Usage.CachedTokens != 800 {
		t.Errorf("expected 800 cached tokens, got %d", state.Usage.CachedTokens)
	}
}
