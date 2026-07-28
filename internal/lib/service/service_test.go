package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/openai/openai-go/v3/responses"
)

func TestParamsFor(t *testing.T) {
	// A prompt far past every ARG_MAX a command line ever had, to pin down that the
	// transport carries it in a body rather than in arguments.
	huge := strings.Repeat("judge this unit. ", 128*1024)
	schema := `{"type":"object","properties":{"ok":{"type":"boolean"}},"required":["ok"],"additionalProperties":false}`

	tests := []struct {
		name    string
		req     Request
		want    string
		wantErr string
	}{
		{
			name: "no effort and no schema sends neither",
			req:  Request{Model: "sonnet", Prompt: "judge this file"},
			want: `{"model":"sonnet","input":"judge this file","instructions":"` + SystemPrompt + `"}`,
		},
		{
			name: "effort low",
			req:  Request{Model: "sonnet", Prompt: "p", Effort: "low"},
			want: `{"model":"sonnet","input":"p","instructions":"` + SystemPrompt + `","reasoning":{"effort":"low"}}`,
		},
		{
			name: "effort medium",
			req:  Request{Model: "sonnet", Prompt: "p", Effort: "medium"},
			want: `{"model":"sonnet","input":"p","instructions":"` + SystemPrompt + `","reasoning":{"effort":"medium"}}`,
		},
		{
			name: "effort high",
			req:  Request{Model: "sonnet", Prompt: "p", Effort: "high"},
			want: `{"model":"sonnet","input":"p","instructions":"` + SystemPrompt + `","reasoning":{"effort":"high"}}`,
		},
		{
			name: "effort xhigh",
			req:  Request{Model: "sonnet", Prompt: "p", Effort: "xhigh"},
			want: `{"model":"sonnet","input":"p","instructions":"` + SystemPrompt + `","reasoning":{"effort":"xhigh"}}`,
		},
		{
			name: "effort max",
			req:  Request{Model: "sonnet", Prompt: "p", Effort: "max"},
			want: `{"model":"sonnet","input":"p","instructions":"` + SystemPrompt + `","reasoning":{"effort":"max"}}`,
		},
		{
			name: "a schema becomes a strict json_schema format",
			req:  Request{Model: "sonnet", Prompt: "p", Schema: json.RawMessage(schema)},
			want: `{"model":"sonnet","input":"p","instructions":"` + SystemPrompt + `","text":{"format":{"type":"json_schema","name":"` + schemaName + `","strict":true,"schema":` + schema + `}}}`,
		},
		{
			name: "an empty schema asks for no particular format",
			req:  Request{Model: "sonnet", Prompt: "p", Schema: json.RawMessage{}},
			want: `{"model":"sonnet","input":"p","instructions":"` + SystemPrompt + `"}`,
		},
		{
			name: "a prompt too large for a command line is carried whole",
			req:  Request{Model: "sonnet", Prompt: huge},
			want: `{"model":"sonnet","input":` + quoted(huge) + `,"instructions":"` + SystemPrompt + `"}`,
		},
		{
			name:    "a schema that is not a JSON object is rejected before any call",
			req:     Request{Model: "sonnet", Prompt: "p", Schema: json.RawMessage(`["not","an","object"]`)},
			wantErr: "schema is not a JSON object",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			params, err := ParamsFor(tt.req)

			if tt.wantErr != "" {
				if err == nil {
					t.Fatalf("ParamsFor(%+v) = %+v, want an error mentioning %q", tt.req, params, tt.wantErr)
				}
				if !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("error = %v, want it to mention %q", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			encoded, err := json.Marshal(params)
			if err != nil {
				t.Fatalf("marshalling the params: %v", err)
			}
			assertSameJSON(t, encoded, tt.want)
		})
	}
}

func TestToken(t *testing.T) {
	tests := []struct {
		name    string
		varName string
		env     map[string]string
		want    string
		wantErr []string
	}{
		{
			name:    "no variable named sends no header at all",
			varName: "",
			env:     map[string]string{"ARITU_TOKEN": "sk-live"},
			want:    "",
		},
		{
			name:    "a named variable resolves to the value in the environment",
			varName: "ARITU_TOKEN",
			env:     map[string]string{"ARITU_TOKEN": "sk-live"},
			want:    "sk-live",
		},
		{
			name:    "a named but unset variable fails naming the key and the variable",
			varName: "ARITU_TOKEN",
			env:     map[string]string{"SOMETHING_ELSE": "sk-live"},
			wantErr: []string{"service.auth_token_var", "$ARITU_TOKEN", "not set"},
		},
		{
			name:    "a named but empty variable fails the same way as an unset one",
			varName: "ARITU_TOKEN",
			env:     map[string]string{"ARITU_TOKEN": ""},
			wantErr: []string{"service.auth_token_var", "$ARITU_TOKEN", "not set"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Token(tt.varName, lookupIn(tt.env))

			if len(tt.wantErr) > 0 {
				if err == nil {
					t.Fatalf("Token(%q) = %q, want an error", tt.varName, got)
				}
				for _, fragment := range tt.wantErr {
					if !strings.Contains(err.Error(), fragment) {
						t.Errorf("error %q does not contain %q", err, fragment)
					}
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Fatalf("Token(%q) = %q, want %q", tt.varName, got, tt.want)
			}
		})
	}
}

func TestAnswerFrom(t *testing.T) {
	tests := []struct {
		name          string
		body          string
		want          string
		wantErr       []string
		wantAnotherGo bool
	}{
		{
			name: "a completed response answers with its output text",
			body: `{"status":"completed","output":[{"type":"message","content":[{"type":"output_text","text":"{\"names\":[\"TestFoo\"]}"}]}]}`,
			want: `{"names":["TestFoo"]}`,
		},
		{
			name: "surrounding whitespace is trimmed off the answer",
			body: `{"status":"completed","output":[{"type":"message","content":[{"type":"output_text","text":"  {\"names\":[]}  "}]}]}`,
			want: `{"names":[]}`,
		},
		{
			name: "text split across content blocks is joined",
			body: `{"status":"completed","output":[{"type":"message","content":[{"type":"output_text","text":"{\"ok\":"},{"type":"output_text","text":"true}"}]}]}`,
			want: `{"ok":true}`,
		},
		{
			name: "an answer wrapped in a code fence is unwrapped",
			body: "{\"status\":\"completed\",\"output\":[{\"type\":\"message\",\"content\":[{\"type\":\"output_text\",\"text\":\"```json\\n{\\\"ok\\\":true}\\n```\"}]}]}",
			want: `{"ok":true}`,
		},
		{
			name:          "an answer of prose is worth another turn and carries what the model said",
			body:          `{"status":"completed","output":[{"type":"message","content":[{"type":"output_text","text":"I looked at the file and it seems fine."}]}]}`,
			wantErr:       []string{"prose instead of JSON", "I looked at the file"},
			wantAnotherGo: true,
		},
		{
			name:          "a failed status is worth another turn and carries the code and message",
			body:          `{"status":"failed","error":{"code":"server_error","message":"upstream exploded"},"output":[]}`,
			wantErr:       []string{"server_error", "upstream exploded"},
			wantAnotherGo: true,
		},
		{
			name:          "a failed status with nothing to report still says so",
			body:          `{"status":"failed","output":[]}`,
			wantErr:       []string{"no detail reported"},
			wantAnotherGo: true,
		},
		{
			name:          "an incomplete status is worth another turn and names the reason",
			body:          `{"status":"incomplete","incomplete_details":{"reason":"max_output_tokens"},"output":[]}`,
			wantErr:       []string{"incomplete", "max_output_tokens"},
			wantAnotherGo: true,
		},
		{
			name:          "an incomplete status without a reason still says so",
			body:          `{"status":"incomplete","output":[]}`,
			wantErr:       []string{"incomplete", "no reason reported"},
			wantAnotherGo: true,
		},
		{
			name:          "a refusal is worth another turn and carries what the model said",
			body:          `{"status":"completed","output":[{"type":"message","content":[{"type":"refusal","refusal":"I cannot help with that"}]}]}`,
			wantErr:       []string{"refused", "I cannot help with that"},
			wantAnotherGo: true,
		},
		{
			name:    "a completed response carrying no text is not worth another turn",
			body:    `{"status":"completed","output":[]}`,
			wantErr: []string{"no output text", "completed"},
		},
		{
			name:    "a cancelled response is not worth another turn",
			body:    `{"status":"cancelled","output":[]}`,
			wantErr: []string{"no output text", "cancelled"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := AnswerFrom(replyFrom(t, tt.body))

			if len(tt.wantErr) > 0 {
				if err == nil {
					t.Fatalf("AnswerFrom() = %s, want an error", got)
				}
				for _, fragment := range tt.wantErr {
					if !strings.Contains(err.Error(), fragment) {
						t.Errorf("error %q does not contain %q", err, fragment)
					}
				}
				if isAnotherGo := errors.Is(err, errModelFailure); isAnotherGo != tt.wantAnotherGo {
					t.Errorf("worth another turn = %v, want %v (error: %v)", isAnotherGo, tt.wantAnotherGo, err)
				}
				if got != nil {
					t.Errorf("want nil output alongside an error, got %s", got)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if string(got) != tt.want {
				t.Fatalf("AnswerFrom() = %s, want %s", got, tt.want)
			}
		})
	}
}

func TestNewSendsTheAuthorizationHeaderOnlyWhenATokenIsConfigured(t *testing.T) {
	tests := []struct {
		name  string
		token string
		want  string
	}{
		{name: "a configured token is sent as a bearer credential", token: "sk-live", want: "Bearer sk-live"},
		{name: "no token sends no authorization header at all", token: "", want: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var got string
			endpoint := serveOneReply(t, `{"status":"completed","output":[{"type":"message","content":[{"type":"output_text","text":"{}"}]}]}`, func(r *http.Request) {
				got = r.Header.Get("Authorization")
			})

			if _, err := New(endpoint, tt.token)(context.Background(), Request{Model: "sonnet", Prompt: "p"}); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if got != tt.want {
				t.Fatalf("Authorization = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestNewSendsTheMappedRequestAndReturnsTheAnswer(t *testing.T) {
	var body []byte
	endpoint := serveOneReply(t, `{"status":"completed","output":[{"type":"message","content":[{"type":"output_text","text":"{\"names\":[\"TestFoo\"]}"}]}]}`, func(r *http.Request) {
		body, _ = io.ReadAll(r.Body)
	})

	got, err := New(endpoint, "")(context.Background(), Request{
		Model:  "sonnet",
		Prompt: "list the tests",
		Effort: "high",
		Schema: json.RawMessage(`{"type":"object","properties":{"names":{"type":"array","items":{"type":"string"}}},"required":["names"],"additionalProperties":false}`),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if string(got) != `{"names":["TestFoo"]}` {
		t.Errorf("answer = %s, want the output text", got)
	}
	assertSameJSON(t, body, `{"model":"sonnet","input":"list the tests","instructions":"`+SystemPrompt+`","reasoning":{"effort":"high"},"text":{"format":{"type":"json_schema","name":"`+schemaName+`","strict":true,"schema":{"type":"object","properties":{"names":{"type":"array","items":{"type":"string"}}},"required":["names"],"additionalProperties":false}}}}`)
}

// TestNewReportsATransportFailureWithoutAskingForAnotherTurn covers the failures
// that reach no model at all. None of them is worth a fresh turn, and none of them
// may take the process down: an endpoint aritu does not own decides these, so they
// have to arrive as values (R-10).
func TestNewReportsATransportFailureWithoutAskingForAnotherTurn(t *testing.T) {
	tests := []struct {
		name     string
		endpoint func(*testing.T) string
		wantErr  string
	}{
		{
			name: "an endpoint that rejects the request",
			endpoint: func(t *testing.T) string {
				return serve(t, func(w http.ResponseWriter, _ *http.Request) {
					w.WriteHeader(http.StatusUnauthorized)
					fmt.Fprint(w, `{"error":{"message":"invalid token","type":"invalid_request_error"}}`)
				})
			},
			wantErr: "401",
		},
		{
			name: "an endpoint nothing is listening on",
			endpoint: func(t *testing.T) string {
				server := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
				address := server.URL + "/"
				server.Close()
				return address
			},
			wantErr: "connect",
		},
		{
			// The SDK decodes into a **Response, so a bare null leaves the pointer
			// nil and reports no error at all. Dereferencing it would panic a whole
			// sweep on one gateway's error page.
			name: "an endpoint answering a JSON null body",
			endpoint: func(t *testing.T) string {
				return serve(t, func(w http.ResponseWriter, _ *http.Request) {
					fmt.Fprint(w, `null`)
				})
			},
			wantErr: "no body",
		},
		{
			name: "an endpoint answering nothing at all",
			endpoint: func(t *testing.T) string {
				return serve(t, func(http.ResponseWriter, *http.Request) {})
			},
			wantErr: "EOF",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			endpoint := tt.endpoint(t)

			_, err := New(endpoint, "")(context.Background(), Request{Model: "sonnet", Prompt: "p"})

			if err == nil {
				t.Fatal("want an error, got none")
			}
			if errors.Is(err, errModelFailure) {
				t.Errorf("a transport failure must not ask for another turn, got %v", err)
			}
			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("error %q does not mention %q", err, tt.wantErr)
			}
			if !strings.Contains(err.Error(), endpoint) {
				t.Errorf("error %q does not name the endpoint %s", err, endpoint)
			}
		})
	}
}

func TestRetry(t *testing.T) {
	answer := json.RawMessage(`{"ok":true}`)
	modelFailure := fmt.Errorf("service: %w: refused", errModelFailure)
	unreachable := errors.New("service: connection refused")

	tests := []struct {
		name      string
		attempts  int
		replies   []error
		wantCalls int
		wantErr   error
	}{
		{
			name:      "an answer on the first turn asks once",
			attempts:  3,
			replies:   []error{nil},
			wantCalls: 1,
		},
		{
			name:      "a model failure is asked again",
			attempts:  3,
			replies:   []error{modelFailure, nil},
			wantCalls: 2,
		},
		{
			name:      "the last failure is reported once the attempts run out",
			attempts:  3,
			replies:   []error{modelFailure, modelFailure, modelFailure},
			wantCalls: 3,
			wantErr:   modelFailure,
		},
		{
			name:      "a failure a second turn would meet again is not retried",
			attempts:  3,
			replies:   []error{unreachable},
			wantCalls: 1,
			wantErr:   unreachable,
		},
		{
			name:      "one attempt leaves the ask alone",
			attempts:  1,
			replies:   []error{modelFailure},
			wantCalls: 1,
			wantErr:   modelFailure,
		},
		{
			name:      "attempts below one leave the ask alone",
			attempts:  0,
			replies:   []error{modelFailure},
			wantCalls: 1,
			wantErr:   modelFailure,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			calls := 0
			scripted := func(context.Context, Request) (json.RawMessage, error) {
				defer func() { calls++ }()
				if calls >= len(tc.replies) {
					t.Fatalf("asked %d times, only %d replies scripted", calls+1, len(tc.replies))
				}
				if err := tc.replies[calls]; err != nil {
					return nil, err
				}
				return answer, nil
			}

			got, err := Retry(scripted, tc.attempts)(context.Background(), Request{})

			if calls != tc.wantCalls {
				t.Errorf("asked %d times, want %d", calls, tc.wantCalls)
			}
			if !errors.Is(err, tc.wantErr) {
				t.Fatalf("error = %v, want %v", err, tc.wantErr)
			}
			if tc.wantErr == nil && string(got) != string(answer) {
				t.Errorf("answer = %s, want %s", got, answer)
			}
		})
	}
}

func TestRetryStopsWhenTheContextEnds(t *testing.T) {
	modelFailure := fmt.Errorf("service: %w: refused", errModelFailure)
	ctx, cancel := context.WithCancel(context.Background())

	calls := 0
	failing := func(context.Context, Request) (json.RawMessage, error) {
		calls++
		cancel()
		return nil, modelFailure
	}

	_, err := Retry(failing, 5)(ctx, Request{})

	if calls != 1 {
		t.Errorf("asked %d times after the context ended, want 1", calls)
	}
	if !errors.Is(err, modelFailure) {
		t.Errorf("error = %v, want the model failure", err)
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

// serveOneReply stands up a real endpoint answering with the given body, so the
// transport is exercised over HTTP rather than described.
func serveOneReply(t *testing.T, body string, observe func(*http.Request)) string {
	t.Helper()
	return serve(t, func(w http.ResponseWriter, r *http.Request) {
		observe(r)
		fmt.Fprint(w, body)
	})
}

func serve(t *testing.T, handle http.HandlerFunc) string {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		handle(w, r)
	}))
	t.Cleanup(server.Close)
	return server.URL + "/"
}

func replyFrom(t *testing.T, body string) responses.Response {
	t.Helper()
	var reply responses.Response
	if err := json.Unmarshal([]byte(body), &reply); err != nil {
		t.Fatalf("decoding the stand-in reply %s: %v", body, err)
	}
	return reply
}

// lookupIn is an environment the test built, standing in for os.LookupEnv without
// standing in for the logic under test.
func lookupIn(env map[string]string) func(string) (string, bool) {
	return func(name string) (string, bool) {
		value, isSet := env[name]
		return value, isSet
	}
}

func assertSameJSON(t *testing.T, got []byte, want string) {
	t.Helper()
	var gotAny, wantAny any
	if err := json.Unmarshal(got, &gotAny); err != nil {
		t.Fatalf("decoding what was produced (%s): %v", got, err)
	}
	if err := json.Unmarshal([]byte(want), &wantAny); err != nil {
		t.Fatalf("decoding what was wanted (%s): %v", want, err)
	}
	if !reflect.DeepEqual(gotAny, wantAny) {
		t.Fatalf("produced %s,\nwant     %s", got, want)
	}
}

func quoted(s string) string {
	encoded, err := json.Marshal(s)
	if err != nil {
		panic(fmt.Sprintf("a string failed to marshal, which its type makes impossible: %v", err))
	}
	return string(encoded)
}
