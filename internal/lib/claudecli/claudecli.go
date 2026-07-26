package claudecli

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os/exec"
	"strings"
)

// Request is one non-interactive call to the Claude CLI.
type Request struct {
	Prompt string
	Model  string
	Effort string
	Schema json.RawMessage
}

// Ask is the seam between domain logic and the Claude CLI process, so callers
// can be exercised against table data instead of a live model.
type Ask func(ctx context.Context, req Request) (json.RawMessage, error)

// SystemPrompt replaces the default Claude Code system prompt. The default one
// costs roughly 60k cached tokens per call and invites tool use; this tool wants
// a single JSON answer and nothing else.
const SystemPrompt = "You are a static-analysis tool. You have no tools and must not attempt to use any. Answer only with the requested JSON."

// Exec returns an Ask that runs the given claude binary, writing the prompt to
// its stdin so that large files never hit ARG_MAX.
func Exec(binary string) Ask {
	return func(ctx context.Context, req Request) (json.RawMessage, error) {
		cmd := exec.CommandContext(ctx, binary, Args(req)...)
		cmd.Stdin = strings.NewReader(req.Prompt)

		var stdout, stderr bytes.Buffer
		cmd.Stdout = &stdout
		cmd.Stderr = &stderr

		runErr := cmd.Run()
		if runErr != nil && ctx.Err() != nil {
			return nil, fmt.Errorf("claudecli: %s: %w", binary, ctx.Err())
		}
		return resultFor(binary, stdout.Bytes(), stderr.String(), runErr)
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

// Args builds the CLI arguments for a request, excluding the prompt.
func Args(req Request) []string {
	args := []string{
		"-p",
		"--model", req.Model,
		"--output-format", "json",
		"--tools", "",
		"--safe-mode",
		"--no-session-persistence",
		"--strict-mcp-config",
		"--system-prompt", SystemPrompt,
	}
	if req.Effort != "" {
		args = append(args, "--effort", req.Effort)
	}
	if len(req.Schema) > 0 {
		args = append(args, "--json-schema", string(req.Schema))
	}
	return args
}

// ParseResult extracts the model's answer from an --output-format json
// envelope, preferring the schema-validated structured_output field and falling
// back to the raw result text.
func ParseResult(stdout []byte) (json.RawMessage, error) {
	var env envelope
	if err := json.Unmarshal(stdout, &env); err != nil {
		return nil, fmt.Errorf("claudecli: malformed response envelope: %w", err)
	}
	if env.IsError {
		return nil, fmt.Errorf("claudecli: %w: %s", errModelFailure, env.failure())
	}
	if hasStructuredOutput(env) {
		return env.StructuredOutput, nil
	}
	if answer := strings.TrimSpace(env.Result); answer != "" {
		return json.RawMessage(answer), nil
	}
	return nil, errors.New("claudecli: response carried neither structured_output nor result")
}

// errModelFailure separates a turn the CLI itself flagged, whose text is worth
// surfacing verbatim, from output this package simply could not parse.
var errModelFailure = errors.New("model reported an error")

// envelope is the --output-format json wrapper. A failed turn is signalled both
// by a non-zero exit and by is_error here, so the fields carrying the reason are
// as load-bearing as the ones carrying the answer.
type envelope struct {
	IsError          bool            `json:"is_error"`
	Result           string          `json:"result"`
	StructuredOutput json.RawMessage `json:"structured_output"`
	TerminalReason   string          `json:"terminal_reason"`
	Errors           []string        `json:"errors"`
}

func (e envelope) failure() string {
	detail := strings.Join(e.Errors, "; ")
	if detail == "" {
		detail = strings.TrimSpace(e.Result)
	}
	if detail == "" {
		detail = "no detail reported"
	}
	if e.TerminalReason == "" {
		return detail
	}
	return fmt.Sprintf("terminal_reason=%s: %s", e.TerminalReason, detail)
}

func hasStructuredOutput(e envelope) bool {
	trimmed := strings.TrimSpace(string(e.StructuredOutput))
	return trimmed != "" && trimmed != "null"
}

func resultFor(binary string, stdout []byte, stderr string, runErr error) (json.RawMessage, error) {
	if runErr == nil {
		return ParseResult(stdout)
	}

	var startErr *exec.Error
	if errors.As(runErr, &startErr) {
		return nil, fmt.Errorf("claudecli: cannot run %q: %w", binary, runErr)
	}
	if _, parseErr := ParseResult(stdout); errors.Is(parseErr, errModelFailure) {
		return nil, parseErr
	}
	return nil, fmt.Errorf("claudecli: %s %w: %s", binary, runErr, describeStderr(stderr))
}

func describeStderr(stderr string) string {
	if trimmed := strings.TrimSpace(stderr); trimmed != "" {
		return trimmed
	}
	return "no stderr output"
}
