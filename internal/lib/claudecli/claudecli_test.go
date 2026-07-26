package claudecli

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestArgs(t *testing.T) {
	base := []string{
		"-p",
		"--model", "sonnet",
		"--output-format", "json",
		"--tools", "",
		"--safe-mode",
		"--no-session-persistence",
		"--strict-mcp-config",
		"--system-prompt", SystemPrompt,
	}
	schema := json.RawMessage(`{"type":"object"}`)

	tests := []struct {
		name string
		req  Request
		want []string
	}{
		{
			name: "no effort no schema",
			req:  Request{Model: "sonnet"},
			want: base,
		},
		{
			name: "prompt stays off the command line",
			req:  Request{Model: "sonnet", Prompt: "judge this file"},
			want: base,
		},
		{
			name: "effort only",
			req:  Request{Model: "sonnet", Effort: "low"},
			want: append(slices.Clone(base), "--effort", "low"),
		},
		{
			name: "schema only",
			req:  Request{Model: "sonnet", Schema: schema},
			want: append(slices.Clone(base), "--json-schema", `{"type":"object"}`),
		},
		{
			name: "effort and schema",
			req:  Request{Model: "sonnet", Effort: "high", Schema: schema},
			want: append(slices.Clone(base), "--effort", "high", "--json-schema", `{"type":"object"}`),
		},
		{
			name: "empty schema is skipped",
			req:  Request{Model: "sonnet", Schema: json.RawMessage{}},
			want: base,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Args(tt.req)
			if !slices.Equal(got, tt.want) {
				t.Fatalf("Args() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestParseResult(t *testing.T) {
	tests := []struct {
		name    string
		stdout  string
		want    string
		wantErr []string
	}{
		{
			name:   "structured output preferred over result",
			stdout: `{"type":"result","subtype":"success","is_error":false,"result":"here you go","structured_output":{"names":["TestFoo"]}}`,
			want:   `{"names":["TestFoo"]}`,
		},
		{
			name:   "falls back to result when structured output absent",
			stdout: `{"type":"result","subtype":"success","is_error":false,"result":"  {\"names\":[\"TestFoo\"]}  "}`,
			want:   `{"names":["TestFoo"]}`,
		},
		{
			name:   "falls back to result when structured output is null",
			stdout: `{"type":"result","subtype":"success","is_error":false,"structured_output":null,"result":"{\"names\":[]}"}`,
			want:   `{"names":[]}`,
		},
		{
			name:   "is_error envelope carries terminal reason and errors",
			stdout: `{"type":"result","subtype":"error_max_structured_output_retries","is_error":true,"terminal_reason":"structured_output_retry_exhausted","errors":["Failed to provide surviving structured output","after 5 attempts"]}`,
			wantErr: []string{
				"terminal_reason=structured_output_retry_exhausted",
				"Failed to provide surviving structured output; after 5 attempts",
			},
		},
		{
			name:    "is_error envelope without errors falls back to result text",
			stdout:  `{"type":"result","subtype":"success","is_error":true,"terminal_reason":"api_error","result":"API Error: 500 overloaded_error"}`,
			wantErr: []string{"terminal_reason=api_error", "API Error: 500 overloaded_error"},
		},
		{
			name:    "is_error envelope with nothing to report",
			stdout:  `{"type":"result","is_error":true}`,
			wantErr: []string{"no detail reported"},
		},
		{
			name:    "malformed json",
			stdout:  `{"type":"result",`,
			wantErr: []string{"malformed response envelope"},
		},
		{
			name:    "empty stdout",
			stdout:  "",
			wantErr: []string{"malformed response envelope"},
		},
		{
			name:    "both fields empty",
			stdout:  `{"type":"result","subtype":"success","is_error":false,"result":"   "}`,
			wantErr: []string{"neither structured_output nor result"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseResult([]byte(tt.stdout))
			assertResult(t, got, err, tt.want, tt.wantErr)
		})
	}
}

func TestResultFor(t *testing.T) {
	success := `{"type":"result","subtype":"success","is_error":false,"structured_output":{"names":["TestFoo"]}}`
	modelFailure := `{"type":"result","subtype":"error_during_execution","is_error":true,"terminal_reason":"api_error","errors":["Overloaded"]}`
	exited := errors.New("exit status 1")

	tests := []struct {
		name    string
		stdout  string
		stderr  string
		runErr  error
		want    string
		wantErr []string
	}{
		{
			name:   "clean exit yields the parsed answer",
			stdout: success,
			want:   `{"names":["TestFoo"]}`,
		},
		{
			name:    "clean exit with unparseable stdout",
			stdout:  "not json",
			wantErr: []string{"malformed response envelope"},
		},
		{
			name:    "binary cannot be started",
			runErr:  &exec.Error{Name: "claude", Err: exec.ErrNotFound},
			wantErr: []string{`cannot run "claude"`, "executable file not found"},
		},
		{
			name:    "non-zero exit surfaces the model's own error text",
			stdout:  modelFailure,
			stderr:  "some noise",
			runErr:  exited,
			wantErr: []string{"terminal_reason=api_error", "Overloaded"},
		},
		{
			name:    "non-zero exit with unparseable stdout reports status and stderr",
			stdout:  "panic: boom",
			stderr:  "  authentication failed  ",
			runErr:  exited,
			wantErr: []string{"exit status 1", "authentication failed"},
		},
		{
			name:    "non-zero exit with a clean envelope reports status",
			stdout:  success,
			runErr:  exited,
			wantErr: []string{"exit status 1", "no stderr output"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := resultFor("claude", []byte(tt.stdout), tt.stderr, tt.runErr)
			assertResult(t, got, err, tt.want, tt.wantErr)
		})
	}
}

func TestExecReportsUnstartableBinary(t *testing.T) {
	_, err := Exec("aritu-no-such-binary")(context.Background(), Request{Model: "sonnet"})
	if err == nil {
		t.Fatal("want an error for a binary that is not on PATH")
	}
	if !strings.Contains(err.Error(), "aritu-no-such-binary") {
		t.Fatalf("error %q does not name the binary", err)
	}
}

func TestExecReportsATimeoutAsSuchRatherThanAsAKilledProcess(t *testing.T) {
	binary := filepath.Join(t.TempDir(), "claude")
	if err := os.WriteFile(binary, []byte("#!/bin/sh\nsleep 30\n"), 0o755); err != nil {
		t.Fatalf("writing stand-in claude: %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()

	_, err := Exec(binary)(ctx, Request{Model: "sonnet"})

	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("error = %v, want one wrapping context.DeadlineExceeded so a hung model reads as a timeout", err)
	}
}

func assertResult(t *testing.T, got json.RawMessage, err error, want string, wantErr []string) {
	t.Helper()

	if len(wantErr) > 0 {
		if err == nil {
			t.Fatalf("want an error, got output %s", got)
		}
		for _, fragment := range wantErr {
			if !strings.Contains(err.Error(), fragment) {
				t.Errorf("error %q does not contain %q", err, fragment)
			}
		}
		if got != nil {
			t.Errorf("want nil output alongside an error, got %s", got)
		}
		return
	}

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(got) != want {
		t.Fatalf("got %s, want %s", got, want)
	}
}

func TestThrottleBoundsCallsInFlight(t *testing.T) {
	tests := []struct {
		name     string
		limit    int
		calls    int
		wantPeak int
	}{
		{name: "a limit of one serialises every call", limit: 1, calls: 6, wantPeak: 1},
		{name: "a limit of three admits three at a time", limit: 3, calls: 9, wantPeak: 3},
		{name: "a limit above the call count admits them all", limit: 8, calls: 4, wantPeak: 4},
		{name: "a limit below one leaves the ask unbounded", limit: 0, calls: 5, wantPeak: 5},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var mu sync.Mutex
			inFlight, peak := 0, 0
			hold := func(context.Context, Request) (json.RawMessage, error) {
				mu.Lock()
				inFlight++
				peak = max(peak, inFlight)
				mu.Unlock()
				time.Sleep(30 * time.Millisecond)
				mu.Lock()
				inFlight--
				mu.Unlock()
				return json.RawMessage(`{}`), nil
			}

			throttled := Throttle(hold, tt.limit)
			var wg sync.WaitGroup
			for range tt.calls {
				wg.Add(1)
				go func() {
					defer wg.Done()
					_, _ = throttled(context.Background(), Request{})
				}()
			}
			wg.Wait()

			mu.Lock()
			defer mu.Unlock()
			if peak != tt.wantPeak {
				t.Errorf("peak calls in flight = %d, want %d", peak, tt.wantPeak)
			}
		})
	}
}

func TestThrottleAbandonsAWaitingCallWhenTheContextEnds(t *testing.T) {
	blocked := make(chan struct{})
	defer close(blocked)
	throttled := Throttle(func(context.Context, Request) (json.RawMessage, error) {
		<-blocked
		return json.RawMessage(`{}`), nil
	}, 1)

	go func() { _, _ = throttled(context.Background(), Request{}) }()
	time.Sleep(20 * time.Millisecond)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	if _, err := throttled(ctx, Request{}); !errors.Is(err, context.Canceled) {
		t.Fatalf("err = %v, want context.Canceled", err)
	}
}

func TestThrottleReleasesTheSlotWhenTheCallFails(t *testing.T) {
	throttled := Throttle(func(context.Context, Request) (json.RawMessage, error) {
		return nil, errors.New("boom")
	}, 1)

	for range 3 {
		if _, err := throttled(context.Background(), Request{}); err == nil {
			t.Fatal("want the underlying error, got none")
		}
	}
}
