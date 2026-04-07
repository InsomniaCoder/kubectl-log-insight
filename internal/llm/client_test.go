package llm_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/InsomniaCoder/kubectl-loginsight/internal/llm"
)

func TestAsk_ReturnsLLMContent(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/chat/completions" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"choices": []map[string]any{
				{"message": map[string]any{"content": "This is the LLM answer."}},
			},
		})
	}))
	defer server.Close()

	client := llm.NewClient(server.URL+"/v1", "testkey", "testmodel")
	result, err := client.Ask("some logs", "what happened?")
	if err != nil {
		t.Fatal(err)
	}
	if result != "This is the LLM answer." {
		t.Errorf("unexpected result: %s", result)
	}
}

func TestAsk_Summarize_WhenNoQuestion(t *testing.T) {
	var capturedBody map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewDecoder(r.Body).Decode(&capturedBody)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"choices": []map[string]any{
				{"message": map[string]any{"content": "Summary here."}},
			},
		})
	}))
	defer server.Close()

	client := llm.NewClient(server.URL+"/v1", "testkey", "testmodel")
	_, err := client.Ask("some logs", "")
	if err != nil {
		t.Fatal(err)
	}

	messages := capturedBody["messages"].([]any)
	systemMsg := messages[0].(map[string]any)
	if systemMsg["role"] != "system" {
		t.Error("expected first message to be system role")
	}
	content := systemMsg["content"].(string)
	if len(content) == 0 {
		t.Error("expected non-empty system prompt")
	}
}

func TestAsk_FallsBackToReasoningContent(t *testing.T) {
	// Qwen3 and other reasoning models return empty content with reasoning_content populated.
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"choices": []map[string]any{
				{"message": map[string]any{
					"content":           "",
					"reasoning_content": "This came from reasoning.",
				}},
			},
		})
	}))
	defer server.Close()

	client := llm.NewClient(server.URL+"/v1", "testkey", "testmodel")
	result, err := client.Ask("some logs", "what happened?")
	if err != nil {
		t.Fatal(err)
	}
	if result != "This came from reasoning." {
		t.Errorf("expected reasoning_content fallback, got: %s", result)
	}
}

func TestAsk_ErrorOnNon200(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "internal error", http.StatusInternalServerError)
	}))
	defer server.Close()

	client := llm.NewClient(server.URL+"/v1", "testkey", "testmodel")
	_, err := client.Ask("logs", "question")
	if err == nil {
		t.Error("expected error for non-200 response")
	}
}
