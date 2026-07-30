package service

import (
	"cmp"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/openai/openai-go/v3/option"
	"github.com/openai/openai-go/v3/packages/param"
	"github.com/openai/openai-go/v3/responses"
	"github.com/openai/openai-go/v3/shared"
)

type Request struct {
	Prompt string
	Model  string
	Effort string
	Schema json.RawMessage
	Kind   Kind
}

type Kind int

const (
	Verdict Kind = iota
	Split
)

type Ask func(ctx context.Context, req Request) (json.RawMessage, error)

const SystemPrompt = "You are a static-analysis tool. You have no tools and must not attempt to use any. Answer only with the requested JSON."

func New(endpoint, token string) Ask {
	client := responses.NewResponseService(clientOptions(endpoint, token)...)
	return func(ctx context.Context, req Request) (json.RawMessage, error) {
		params, err := ParamsFor(req)
		if err != nil {
			return nil, err
		}
		reply, err := client.New(ctx, params)
		if err != nil {
			return nil, fmt.Errorf("service: %s: %w", endpoint, err)
		}
		if reply == nil {
			return nil, fmt.Errorf("service: %s: response carried no body", endpoint)
		}
		return AnswerFrom(*reply)
	}
}

func Token(varName string, lookup func(string) (string, bool)) (string, error) {
	if varName == "" {
		return "", nil
	}
	value, isSet := lookup(varName)
	if !isSet || value == "" {
		return "", fmt.Errorf("service.auth_token_var names $%s, which is not set", varName)
	}
	return value, nil
}

func ParamsFor(req Request) (responses.ResponseNewParams, error) {
	params := responses.ResponseNewParams{
		Model:        req.Model,
		Instructions: param.NewOpt(SystemPrompt),
		Input:        responses.ResponseNewParamsInputUnion{OfString: param.NewOpt(req.Prompt)},
	}
	if req.Effort != "" {
		params.Reasoning = shared.ReasoningParam{Effort: shared.ReasoningEffort(req.Effort)}
	}
	if len(req.Schema) == 0 {
		return params, nil
	}
	format, err := formatFor(req.Schema)
	if err != nil {
		return responses.ResponseNewParams{}, err
	}
	params.Text = responses.ResponseTextConfigParam{Format: format}
	return params, nil
}

func AnswerFrom(reply responses.Response) (json.RawMessage, error) {
	if refusal := findRefusal(reply); refusal != "" {
		return nil, fmt.Errorf("service: %w: refused: %s", errModelFailure, refusal)
	}
	if isRetryableStatus[reply.Status] {
		return nil, fmt.Errorf("service: %w: %s", errModelFailure, describeFailure(reply))
	}
	answer := strings.TrimSpace(reply.OutputText())
	if answer == "" {
		return nil, fmt.Errorf("service: response carried no output text, and its status was %q", reply.Status)
	}
	answer = unfenced(answer)
	if !json.Valid([]byte(answer)) {
		return nil, fmt.Errorf("service: %w: answered prose instead of JSON: %s", errModelFailure, snippetOf(answer))
	}
	return json.RawMessage(answer), nil
}

// The SDK already paces 408, 409, 429 and 5xx with backoff of its own.
func Retry(ask Ask, attempts int) Ask {
	if attempts < 2 {
		return ask
	}
	return func(ctx context.Context, req Request) (json.RawMessage, error) {
		var err error
		for range attempts {
			var answer json.RawMessage
			answer, err = ask(ctx, req)
			if err == nil {
				return answer, nil
			}
			if ctx.Err() != nil || !errors.Is(err, errModelFailure) {
				return nil, err
			}
		}
		return nil, err
	}
}

func Throttle(ask Ask, limit int) Ask {
	if limit < 1 {
		return ask
	}
	slots := make(chan struct{}, limit)
	return func(ctx context.Context, req Request) (json.RawMessage, error) {
		select {
		case slots <- struct{}{}:
		case <-ctx.Done():
			return nil, ctx.Err()
		}
		defer func() { <-slots }()
		return ask(ctx, req)
	}
}

var errModelFailure = errors.New("model reported an error")

// The API requires the response format to carry a name.
const schemaName = "answer"

var isRetryableStatus = map[responses.ResponseStatus]bool{
	responses.ResponseStatusFailed:     true,
	responses.ResponseStatusIncomplete: true,
}

func clientOptions(endpoint, token string) []option.RequestOption {
	opts := []option.RequestOption{option.WithBaseURL(endpoint)}
	if token != "" {
		opts = append(opts, option.WithHeader("Authorization", "Bearer "+token))
	}
	return opts
}

// A json_schema format the endpoint enforces rather than suggests requires
// strict, and the generated verdict schema depends on that enforcement.
func formatFor(schema json.RawMessage) (responses.ResponseFormatTextConfigUnionParam, error) {
	var decoded map[string]any
	if err := json.Unmarshal(schema, &decoded); err != nil {
		return responses.ResponseFormatTextConfigUnionParam{}, fmt.Errorf("service: schema is not a JSON object: %w", err)
	}
	format := responses.ResponseFormatTextConfigParamOfJSONSchema(schemaName, decoded)
	format.OfJSONSchema.Strict = param.NewOpt(true)
	return format, nil
}

// unfenced strips the Markdown fence off an answer that is one whole fenced
// block. The format asks for bare JSON, but an endpoint that merely relays a
// model's text can deliver the same value wrapped, and the wrapping is not the
// model changing its answer.
func unfenced(text string) string {
	if !strings.HasPrefix(text, "```") || !strings.HasSuffix(text, "```") {
		return text
	}
	body := strings.TrimSuffix(text, "```")
	if _, after, hasBreak := strings.Cut(body, "\n"); hasBreak {
		return strings.TrimSpace(after)
	}
	return text
}

func snippetOf(text string) string {
	const ceiling = 120
	flattened := strings.Join(strings.Fields(text), " ")
	if len(flattened) <= ceiling {
		return flattened
	}
	return flattened[:ceiling] + "…"
}

// findRefusal reports the refusal text the model gave, if it gave one. A refusal
// is a content item beside the output text rather than a status, and OutputText
// skips it, so a refused call otherwise reads as an empty answer.
func findRefusal(reply responses.Response) string {
	for _, item := range reply.Output {
		for _, content := range item.Content {
			if content.Type == "refusal" {
				return content.Refusal
			}
		}
	}
	return ""
}

func describeFailure(reply responses.Response) string {
	if reply.Status == responses.ResponseStatusIncomplete {
		return fmt.Sprintf("incomplete: %s", cmp.Or(reply.IncompleteDetails.Reason, "no reason reported"))
	}
	detail := cmp.Or(strings.TrimSpace(reply.Error.Message), "no detail reported")
	if code := string(reply.Error.Code); code != "" {
		return fmt.Sprintf("%s: %s", code, detail)
	}
	return detail
}
