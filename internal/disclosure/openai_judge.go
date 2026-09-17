package disclosure

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/openai/openai-go"
	"github.com/openai/openai-go/option"
	"github.com/openai/openai-go/shared"
)

const systemPrompt = `You read the title and description of a software pull request and decide whether the author states that an AI model, coding assistant, or agent helped produce the change or the description. A statement that no AI was used counts as not disclosed. A mention of an AI product as the subject of the code, such as fixing a bug in an AI client library, counts as not disclosed. Answer with JSON. When the author does disclose AI help, copy the single sentence that says so exactly as it appears.`

const maxBodyRunes = 12000

// maxAnswerTokens bounds the answer. A verdict is two fields and a
// quoted sentence, so an answer that hits the bound was not an answer.
const maxAnswerTokens = 300

// ErrTruncatedAnswer reports that the model hit the token bound before
// finishing its answer.
var ErrTruncatedAnswer = errors.New("judge answer was cut off before it finished")

// noThinking is passed to the chat template so a thinking model such as
// Qwen3.5 answers directly. Left on, a 0.8B model reasons until the
// context fills and the answer never starts. Servers whose templates
// have no such switch ignore the argument.
var noThinking = map[string]any{"enable_thinking": false}

var responseSchema = map[string]any{
	"type": "object",
	"properties": map[string]any{
		"disclosed": map[string]any{"type": "boolean"},
		"quote":     map[string]any{"type": "string"},
	},
	"required":             []string{"disclosed", "quote"},
	"additionalProperties": false,
}

// OpenAIJudge asks an OpenAI-compatible chat endpoint for a verdict and
// verifies the quoted sentence against the overview. CallTimeout bounds
// each Judge call on its own, so a slow model fails the disclosure
// signal alone and leaves the rest of the run untouched. Zero means no
// bound beyond the caller's context.
type OpenAIJudge struct {
	client      openai.Client
	model       string
	CallTimeout time.Duration
}

// NewOpenAIJudge returns a judge that talks to baseURL, which ends in
// /v1 for a llama.cpp server or an OpenAI-compatible gateway. An empty
// apiKey sends no Authorization header.
func NewOpenAIJudge(baseURL, apiKey, model string) (*OpenAIJudge, error) {
	if strings.TrimSpace(baseURL) == "" {
		return nil, errors.New("judge base URL must not be empty")
	}
	if strings.TrimSpace(model) == "" {
		return nil, errors.New("judge model must not be empty")
	}
	opts := []option.RequestOption{option.WithBaseURL(baseURL)}
	if apiKey != "" {
		opts = append(opts, option.WithAPIKey(apiKey))
	}
	return &OpenAIJudge{client: openai.NewClient(opts...), model: model}, nil
}

// WaitReady polls the endpoint's model list until it answers or the
// context ends. A llama.cpp server refuses requests while it loads the
// model, so callers that launched the server wait here first.
func (j *OpenAIJudge) WaitReady(ctx context.Context) error {
	delay := 250 * time.Millisecond
	for {
		_, err := j.client.Models.List(ctx)
		if err == nil {
			return nil
		}
		select {
		case <-ctx.Done():
			return fmt.Errorf("judge not ready: %w", errors.Join(ctx.Err(), err))
		case <-time.After(delay):
		}
		if delay < 2*time.Second {
			delay *= 2
		}
	}
}

// Judge sends the overview and returns the verified verdict. A verdict
// whose quote is absent from the overview returns ErrUnverifiedQuote.
func (j *OpenAIJudge) Judge(ctx context.Context, o Overview) (Verdict, error) {
	if j.CallTimeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(context.WithoutCancel(ctx), j.CallTimeout)
		defer cancel()
	}
	body := o.Body
	if runes := []rune(body); len(runes) > maxBodyRunes {
		body = string(runes[:maxBodyRunes])
	}
	user := fmt.Sprintf("Title: %s\n\nDescription:\n%s", o.Title, body)
	params := openai.ChatCompletionNewParams{
		Model:               j.model,
		Temperature:         openai.Float(0),
		MaxCompletionTokens: openai.Int(maxAnswerTokens),
		Messages: []openai.ChatCompletionMessageParamUnion{
			openai.SystemMessage(systemPrompt),
			openai.UserMessage(user),
		},
		ResponseFormat: openai.ChatCompletionNewParamsResponseFormatUnion{
			OfJSONSchema: &shared.ResponseFormatJSONSchemaParam{
				JSONSchema: shared.ResponseFormatJSONSchemaJSONSchemaParam{
					Name:   "disclosure_verdict",
					Schema: responseSchema,
					Strict: openai.Bool(true),
				},
			},
		},
	}
	resp, err := j.client.Chat.Completions.New(ctx, params, option.WithJSONSet("chat_template_kwargs", noThinking))
	if err != nil {
		return Verdict{}, fmt.Errorf("chat completion: %w", err)
	}
	if len(resp.Choices) == 0 {
		return Verdict{}, errors.New("chat completion returned no choices")
	}
	if resp.Choices[0].FinishReason == "length" {
		return Verdict{}, ErrTruncatedAnswer
	}
	var out struct {
		Disclosed bool   `json:"disclosed"`
		Quote     string `json:"quote"`
	}
	if err := json.Unmarshal([]byte(resp.Choices[0].Message.Content), &out); err != nil {
		return Verdict{}, fmt.Errorf("decode verdict: %w", err)
	}
	if !out.Disclosed {
		return Verdict{}, nil
	}
	if !containsNormalized(o.Title+"\n"+o.Body, out.Quote) {
		return Verdict{}, ErrUnverifiedQuote
	}
	return Verdict{Disclosed: true, Quote: strings.TrimSpace(out.Quote)}, nil
}
