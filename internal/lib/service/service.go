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

// Request is one non-interactive call to a Responses-compatible endpoint.
type Request struct {
	Prompt string
	Model  string
	Effort string
	Schema json.RawMessage
}

// Ask is the seam between domain logic and the model endpoint, so callers can be
// exercised against table data instead of a live model.
type Ask func(ctx context.Context, req Request) (json.RawMessage, error)

// SystemPrompt is what keeps the reply to bare JSON and off tool use.
const SystemPrompt = "You are a static-analysis tool. You have no tools and must not attempt to use any. Answer only with the requested JSON."

// New returns an Ask that calls the Responses API at endpoint. An empty token
// sends no Authorization header at all, which is what a local endpoint that
// ignores auth wants.
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

// Token resolves the bearer token from the name of an environment variable. The
// config field holds a name rather than a secret, so the lookup is passed in and
// the environment is read at the boundary.
//
// No name means no auth, which is not an error. A name whose variable is unset or
// empty is, because the alternative is paying for the typo as a wall of 401s from
// every call in the sweep, arriving minutes later.
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

// ParamsFor maps a Request onto the Responses parameters. Effort is omitted
// entirely when empty rather than sent as a blank string, and a request carrying
// no schema asks for no particular format.
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

// AnswerFrom reads the model's answer out of a response. With a strict
// json_schema format the output text is the structured value, so there is nothing
// to unwrap.
//
// A refusal, a run the endpoint marked failed or incomplete, and an answer that
// is not JSON are the conditions a fresh turn can fix, and only they are
// reported as retryable.
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

// Retry runs a call again when the model failed to answer in the shape it was
// asked for. This starts a fresh turn, which is what recovers a call whose whole
// context went wrong rather than one unlucky sample.
//
// Only a failure the model itself reported is retried. A refused connection, a
// rejected request and a cancelled context are conditions a second attempt would
// meet again, and retrying them would turn a clear error into a slow one. The
// SDK already paces 408, 409, 429 and 5xx with backoff, which is a different
// failure from this one. Attempts below two leave the ask untouched.
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

// Throttle bounds how many calls may be in flight at once. Fixture-level and
// vote-level concurrency multiply, so a ceiling at this seam is the only one that
// holds however the callers above it nest their goroutines. A limit below one
// leaves the ask unbounded.
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

// errModelFailure separates a turn the endpoint itself flagged, whose text is
// worth surfacing verbatim, from a transport that never reached a model.
var errModelFailure = errors.New("model reported an error")

// schemaName labels the response format. The API requires a name and nothing
// reads it back, so one constant serves both call sites rather than a field on
// Request that every caller would have to fill in.
const schemaName = "answer"

// isRetryableStatus is the whole of the retry classification: a run that failed
// or ran out of room is worth another turn, and every other status is either an
// answer or a condition a second turn would meet again.
var isRetryableStatus = map[responses.ResponseStatus]bool{
	responses.ResponseStatusFailed:     true,
	responses.ResponseStatusIncomplete: true,
}

// clientOptions omits the Authorization header entirely when no token was
// configured. WithAPIKey is deliberately not used: it is one header spelled a
// particular way, and two ways to set Authorization is a precedence question
// nobody needs to answer.
func clientOptions(endpoint, token string) []option.RequestOption {
	opts := []option.RequestOption{option.WithBaseURL(endpoint)}
	if token != "" {
		opts = append(opts, option.WithHeader("Authorization", "Bearer "+token))
	}
	return opts
}

// formatFor turns the schema a caller generated into a strict json_schema format,
// so the endpoint enforces the schema rather than merely suggesting it — that is
// the property the generated verdict schema depends on for uniqueness and
// completeness of keys.
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

// snippetOf keeps a surfaced answer to one readable line, so a page of prose
// does not become a page of error.
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
