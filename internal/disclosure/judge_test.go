package disclosure

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOpenAIJudge_VerifiesQuoteAgainstOverview(t *testing.T) {
	tests := []struct {
		name    string
		answer  map[string]any
		want    Verdict
		wantErr error
	}{
		{
			name:   "quote with whitespace and case drift still verifies",
			answer: map[string]any{"disclosed": true, "quote": "i drafted  this with claude and reviewed every line."},
			want:   Verdict{Disclosed: true, Quote: "i drafted  this with claude and reviewed every line."},
		},
		{
			name:    "fabricated quote is rejected",
			answer:  map[string]any{"disclosed": true, "quote": "Generated entirely by GPT."},
			wantErr: ErrUnverifiedQuote,
		},
		{
			name:   "not disclosed",
			answer: map[string]any{"disclosed": false, "quote": ""},
			want:   Verdict{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := fakeChatServer(t, tt.answer)
			j, err := NewOpenAIJudge(srv.URL+"/v1", "", "local")
			require.NoError(t, err)
			got, err := j.Judge(context.Background(), Overview{
				Title: "fix: tighten parser",
				Body:  "The parser accepted a trailing comma.\n\nI drafted this with Claude and reviewed every line.",
			})
			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestOpenAIJudge_TruncatedAnswerIsAnError(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("POST /v1/chat/completions", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"c","object":"chat.completion","created":1,"model":"local","choices":[{"index":0,"finish_reason":"length","message":{"role":"assistant","content":""}}]}`))
	})
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)

	j, err := NewOpenAIJudge(srv.URL+"/v1", "", "local")
	require.NoError(t, err)
	_, err = j.Judge(context.Background(), Overview{Title: "t", Body: "b"})
	assert.ErrorIs(t, err, ErrTruncatedAnswer)
}

func TestOpenAIJudge_RequestCarriesSchemaCapAndNoThinking(t *testing.T) {
	var seen atomic.Pointer[map[string]any]
	mux := http.NewServeMux()
	mux.HandleFunc("POST /v1/chat/completions", func(w http.ResponseWriter, r *http.Request) {
		var req map[string]any
		require.NoError(t, json.NewDecoder(r.Body).Decode(&req))
		seen.Store(&req)
		writeChat(t, w, map[string]any{"disclosed": false, "quote": ""})
	})
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)

	j, err := NewOpenAIJudge(srv.URL+"/v1", "", "local")
	require.NoError(t, err)
	_, err = j.Judge(context.Background(), Overview{Title: "t", Body: "b"})
	require.NoError(t, err)

	req := *seen.Load()
	assert.Equal(t, float64(0), req["temperature"])
	assert.Equal(t, float64(maxAnswerTokens), req["max_completion_tokens"])
	assert.Equal(t, map[string]any{"enable_thinking": false}, req["chat_template_kwargs"])
	format, ok := req["response_format"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "json_schema", format["type"])
}

func TestOpenAIJudge_WaitReadyRetriesUntilModelsAnswer(t *testing.T) {
	var calls atomic.Int32
	mux := http.NewServeMux()
	mux.HandleFunc("GET /v1/models", func(w http.ResponseWriter, _ *http.Request) {
		if calls.Add(1) < 3 {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"object":"list","data":[{"id":"local","object":"model"}]}`))
	})
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)

	j, err := NewOpenAIJudge(srv.URL+"/v1", "", "local")
	require.NoError(t, err)
	require.NoError(t, j.WaitReady(context.Background()))
	assert.GreaterOrEqual(t, calls.Load(), int32(3))
}

func fakeChatServer(t *testing.T, answer map[string]any) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()
	mux.HandleFunc("POST /v1/chat/completions", func(w http.ResponseWriter, _ *http.Request) {
		writeChat(t, w, answer)
	})
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return srv
}

func writeChat(t *testing.T, w http.ResponseWriter, answer map[string]any) {
	t.Helper()
	content, err := json.Marshal(answer)
	require.NoError(t, err)
	w.Header().Set("Content-Type", "application/json")
	require.NoError(t, json.NewEncoder(w).Encode(map[string]any{
		"id":      "chatcmpl-1",
		"object":  "chat.completion",
		"created": 1,
		"model":   "local",
		"choices": []map[string]any{{
			"index":         0,
			"finish_reason": "stop",
			"message":       map[string]any{"role": "assistant", "content": string(content)},
		}},
	}))
}
