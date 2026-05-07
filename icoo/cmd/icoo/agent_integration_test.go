package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunAgentProjectPersistsSessionWithToolTurns(t *testing.T) {
	agentRoot, err := filepath.Abs(filepath.Join("..", "..", "apps", "agent"))
	if err != nil {
		t.Fatalf("resolve agent root: %v", err)
	}

	requestCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/chat/completions" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		defer r.Body.Close()

		var payload map[string]any
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatalf("decode request: %v", err)
		}

		requestCount++
		messages, _ := payload["messages"].([]any)
		if requestCount == 1 {
			writeJSON(t, w, map[string]any{
				"choices": []any{
					map[string]any{
						"finish_reason": "tool_calls",
						"message": map[string]any{
							"role":              "assistant",
							"content":           "",
							"reasoning_content": "inspect root first",
							"tool_calls": []any{
								map[string]any{
									"id":   "call_1",
									"type": "function",
									"function": map[string]any{
										"name":      "listDir",
										"arguments": "{\"path\":\"@/\"}",
									},
								},
							},
						},
					},
				},
			})
			return
		}

		if len(messages) < 4 {
			t.Fatalf("expected replayed messages, got %d", len(messages))
		}
		assistant, _ := messages[2].(map[string]any)
		if assistant["reasoning_content"] != "inspect root first" {
			t.Fatalf("expected reasoning_content replay, got %#v", assistant["reasoning_content"])
		}
		toolMsg, _ := messages[3].(map[string]any)
		if toolMsg["role"] != "tool" {
			t.Fatalf("expected tool replay message, got %#v", toolMsg["role"])
		}

		writeJSON(t, w, map[string]any{
			"choices": []any{
				map[string]any{
					"finish_reason": "stop",
					"message": map[string]any{
						"role":              "assistant",
						"content":           "done",
						"reasoning_content": "final answer",
					},
				},
			},
		})
	}))
	defer server.Close()

	stateDir := t.TempDir()
	sessionDir := filepath.Join(stateDir, "sessions")
	sessionID := "s_cli_agent_test"
	if err := runProjectPath(agentRoot, []string{
		"--workspace", agentRoot,
		"--session", sessionID,
		"--session-dir", sessionDir,
		"--log-path", filepath.Join(stateDir, "agent.log"),
		"--base-url", server.URL,
		"--api-key", "test-key",
		"--model", "mock-model",
		"--task", "List the root and summarize.",
		"--max-turns", "3",
		"--max-files", "4",
		"--max-total-bytes", "2048",
		"--stream-final-answer", "false",
	}); err != nil {
		t.Fatalf("expected agent project run to succeed, got: %v", err)
	}

	if requestCount != 2 {
		t.Fatalf("expected exactly 2 model requests, got %d", requestCount)
	}

	sessionPath := filepath.Join(sessionDir, sessionID+".json")
	data, err := os.ReadFile(sessionPath)
	if err != nil {
		t.Fatalf("read session file: %v", err)
	}

	var session map[string]any
	if err := json.Unmarshal(data, &session); err != nil {
		t.Fatalf("decode session file: %v", err)
	}
	if version, ok := session["sessionSchemaVersion"].(float64); !ok || int(version) != 2 {
		t.Fatalf("expected session schema version 2, got %#v", session["sessionSchemaVersion"])
	}
	turns, _ := session["turns"].([]any)
	if len(turns) < 6 {
		t.Fatalf("expected persisted turns, got %d", len(turns))
	}

	foundAssistantToolCall := false
	foundToolMessage := false
	for _, rawTurn := range turns {
		turn, _ := rawTurn.(map[string]any)
		kind, _ := turn["kind"].(string)
		payload, _ := turn["payload"].(map[string]any)
		if kind == "assistant_message" && payload["reasoningContent"] == "inspect root first" {
			foundAssistantToolCall = true
		}
		if kind == "tool_message" {
			content, _ := payload["name"].(string)
			if strings.TrimSpace(content) == "listDir" {
				foundToolMessage = true
			}
		}
	}
	if !foundAssistantToolCall {
		t.Fatal("expected assistant_message with persisted reasoningContent")
	}
	if !foundToolMessage {
		t.Fatal("expected persisted tool_message for listDir")
	}
}

func TestRunAgentProjectStreamsFinalAnswerAfterTools(t *testing.T) {
	agentRoot, err := filepath.Abs(filepath.Join("..", "..", "apps", "agent"))
	if err != nil {
		t.Fatalf("resolve agent root: %v", err)
	}

	requestCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/chat/completions" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		defer r.Body.Close()

		var payload map[string]any
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatalf("decode request: %v", err)
		}

		requestCount++
		if requestCount == 1 {
			writeJSON(t, w, map[string]any{
				"choices": []any{
					map[string]any{
						"finish_reason": "tool_calls",
						"message": map[string]any{
							"role":              "assistant",
							"content":           "",
							"reasoning_content": "inspect root first",
							"tool_calls": []any{
								map[string]any{
									"id":   "call_1",
									"type": "function",
									"function": map[string]any{
										"name":      "listDir",
										"arguments": "{\"path\":\"@/\"}",
									},
								},
							},
						},
					},
				},
			})
			return
		}

		if payload["stream"] != true {
			t.Fatalf("expected final answer request to enable streaming, got %#v", payload["stream"])
		}
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = fmt.Fprint(w, "data: {\"choices\":[{\"delta\":{\"content\":\"hello \"},\"finish_reason\":null}]}\n\n")
		_, _ = fmt.Fprint(w, "data: {\"choices\":[{\"delta\":{\"content\":\"world\"},\"finish_reason\":null}]}\n\n")
		_, _ = fmt.Fprint(w, "data: {\"choices\":[{\"delta\":{},\"finish_reason\":\"stop\"}]}\n\n")
		_, _ = fmt.Fprint(w, "data: [DONE]\n\n")
	}))
	defer server.Close()

	stateDir := t.TempDir()
	sessionDir := filepath.Join(stateDir, "sessions")
	sessionID := "s_cli_agent_stream_test"
	if err := runProjectPath(agentRoot, []string{
		"--workspace", agentRoot,
		"--session", sessionID,
		"--session-dir", sessionDir,
		"--log-path", filepath.Join(stateDir, "agent.log"),
		"--base-url", server.URL,
		"--api-key", "test-key",
		"--model", "mock-model",
		"--task", "List the root and summarize.",
		"--max-turns", "3",
		"--max-files", "4",
		"--max-total-bytes", "2048",
		"--stream-final-answer", "true",
	}); err != nil {
		t.Fatalf("expected agent project run with streaming to succeed, got: %v", err)
	}

	if requestCount != 2 {
		t.Fatalf("expected exactly 2 model requests, got %d", requestCount)
	}

	sessionPath := filepath.Join(sessionDir, sessionID+".json")
	data, err := os.ReadFile(sessionPath)
	if err != nil {
		t.Fatalf("read session file: %v", err)
	}

	var session map[string]any
	if err := json.Unmarshal(data, &session); err != nil {
		t.Fatalf("decode session file: %v", err)
	}
	turns, _ := session["turns"].([]any)
	foundFinalText := false
	for _, rawTurn := range turns {
		turn, _ := rawTurn.(map[string]any)
		if turn["kind"] != "assistant_message" {
			continue
		}
		payload, _ := turn["payload"].(map[string]any)
		if payload["content"] == "hello world" {
			foundFinalText = true
		}
	}
	if !foundFinalText {
		t.Fatal("expected streamed final answer to be persisted as complete assistant_message")
	}
}

func writeJSON(t *testing.T, w http.ResponseWriter, payload map[string]any) {
	t.Helper()
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(payload); err != nil {
		t.Fatalf("encode response: %v", err)
	}
}
