package selftest

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/matthijn/aritu/internal/domain/lint"
	"github.com/matthijn/aritu/internal/domain/rule"
	"github.com/matthijn/aritu/internal/lib/service"
)

func TestHolds(t *testing.T) {
	tests := []struct {
		name   string
		expect rule.Expectation
		counts map[string]int
		votes  int
		want   bool
	}{
		{"pass fixture at votes", rule.ExpectPass, map[string]int{"TestA": 4, "TestB": 4}, 4, true},
		{"pass fixture one vote short", rule.ExpectPass, map[string]int{"TestA": 4, "TestB": 3}, 4, false},
		{"pass fixture at zero", rule.ExpectPass, map[string]int{"TestA": 0, "TestB": 0}, 4, false},
		{"fail fixture at zero", rule.ExpectFail, map[string]int{"TestA": 0, "TestB": 0}, 4, true},
		{"fail fixture with one dissenting vote", rule.ExpectFail, map[string]int{"TestA": 1, "TestB": 0}, 4, false},
		{"fail fixture at votes", rule.ExpectFail, map[string]int{"TestA": 4, "TestB": 4}, 4, false},
		{"pass fixture with no tests", rule.ExpectPass, map[string]int{}, 4, false},
		{"fail fixture with no tests", rule.ExpectFail, map[string]int{}, 4, false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := Holds(tc.expect, tc.counts, tc.votes); got != tc.want {
				t.Fatalf("Holds(%s, %v, %d) = %t, want %t", tc.expect, tc.counts, tc.votes, got, tc.want)
			}
		})
	}
}

func TestHoldsPanicsOnUnknownExpectation(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Fatal("Holds did not panic on an unknown expectation")
		}
	}()
	Holds(rule.Expectation(99), map[string]int{"TestA": 1}, 4)
}

func TestExitFor(t *testing.T) {
	tests := []struct {
		name    string
		results []Result
		want    lint.Exit
	}{
		{"empty table", nil, lint.ExitPass},
		{"every fixture holds", []Result{resultOf("pass-a", true, nil), resultOf("fail-b", true, nil)}, lint.ExitPass},
		{"one fixture misses", []Result{resultOf("pass-a", true, nil), resultOf("fail-b", false, nil)}, lint.ExitFail},
		{"error outranks an earlier miss", []Result{resultOf("pass-a", false, nil), resultOf("fail-b", false, errors.New("unreachable"))}, lint.ExitError},
		{"error outranks a later miss", []Result{resultOf("pass-a", false, errors.New("unreachable")), resultOf("fail-b", false, nil)}, lint.ExitError},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := ExitFor(tc.results); got != tc.want {
				t.Fatalf("ExitFor = %d, want %d", got, tc.want)
			}
		})
	}
}

func TestFormat(t *testing.T) {
	tests := []struct {
		name    string
		opts    Options
		results []Result
		want    string
	}{
		{
			name: "mixed table with a missed and an errored fixture",
			opts: Options{Rule: rule.Rule{Name: "one-reason-to-fail"}, Votes: 4, Model: "sonnet"},
			results: []Result{
				{
					Fixture:  rule.Fixture{Name: "fail-two-behaviors", Expect: rule.ExpectFail},
					Report:   lint.Report{Verdicts: map[string]int{"TestB": 0, "TestA": 0}},
					Held:     true,
					Duration: 1500 * time.Millisecond,
				},
				{
					Fixture:  rule.Fixture{Name: "pass-single-assert", Expect: rule.ExpectPass},
					Report:   lint.Report{Verdicts: map[string]int{"TestA": 3}},
					Duration: 820 * time.Millisecond,
				},
				{
					Fixture:  rule.Fixture{Name: "pass-unreadable", Expect: rule.ExpectPass},
					Err:      errors.New("read testdata: nope"),
					Duration: 12 * time.Millisecond,
				},
			},
			want: "rule: one-reason-to-fail  model: sonnet  votes: 4\n" +
				"\n" +
				"FIXTURE             EXPECT  RESULT  TIME   VERDICTS\n" +
				"fail-two-behaviors  fail    hold    1.5s   TestA=0 TestB=0\n" +
				"pass-single-assert  pass    MISS    820ms  TestA=3\n" +
				"pass-unreadable     pass    ERROR   12ms   read testdata: nope\n" +
				"\n" +
				"1/3 fixtures hold in 2.4s\n",
		},
		{
			name: "every fixture holds",
			opts: Options{Rule: rule.Rule{Name: "named-for-behavior"}, Votes: 2, Model: "haiku"},
			results: []Result{
				{
					Fixture:  rule.Fixture{Name: "pass-alpha", Expect: rule.ExpectPass},
					Report:   lint.Report{Verdicts: map[string]int{"TestAlpha": 2}},
					Held:     true,
					Duration: 1500 * time.Millisecond,
				},
				{
					Fixture:  rule.Fixture{Name: "fail-beta", Expect: rule.ExpectFail},
					Report:   lint.Report{Verdicts: map[string]int{"TestBeta": 0}},
					Held:     true,
					Duration: 1500 * time.Millisecond,
				},
			},
			want: "rule: named-for-behavior  model: haiku  votes: 2\n" +
				"\n" +
				"FIXTURE     EXPECT  RESULT  TIME  VERDICTS\n" +
				"pass-alpha  pass    hold    1.5s  TestAlpha=2\n" +
				"fail-beta   fail    hold    1.5s  TestBeta=0\n" +
				"\n" +
				"2/2 fixtures hold in 2.4s\n",
		},
		{
			name: "an error carrying newlines stays on a single row",
			opts: Options{Rule: rule.Rule{Name: "one-reason-to-fail"}, Votes: 2, Model: "sonnet"},
			results: []Result{
				{
					Fixture:  rule.Fixture{Name: "pass-alpha", Expect: rule.ExpectPass},
					Report:   lint.Report{Verdicts: map[string]int{"TestAlpha": 2}},
					Held:     true,
					Duration: 1500 * time.Millisecond,
				},
				{
					Fixture:  rule.Fixture{Name: "pass-crashed", Expect: rule.ExpectPass},
					Err:      errors.New("service: http://gateway.internal/v1/:\nPOST \"http://gateway.internal/v1/responses\": 503\n    Service Unavailable"),
					Duration: 40 * time.Millisecond,
				},
			},
			want: "rule: one-reason-to-fail  model: sonnet  votes: 2\n" +
				"\n" +
				"FIXTURE       EXPECT  RESULT  TIME  VERDICTS\n" +
				"pass-alpha    pass    hold    1.5s  TestAlpha=2\n" +
				"pass-crashed  pass    ERROR   40ms  service: http://gateway.internal/v1/: POST \"http://gateway.internal/v1/responses\": 503 Service Unavailable\n" +
				"\n" +
				"1/2 fixtures hold in 2.4s\n",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var out bytes.Buffer
			if err := Format(&out, tc.opts, tc.results, 2400*time.Millisecond); err != nil {
				t.Fatalf("Format returned %v", err)
			}
			if out.String() != tc.want {
				t.Fatalf("Format wrote:\n%s\nwant:\n%s", out.String(), tc.want)
			}
		})
	}
}

func TestFormatReturnsWriterError(t *testing.T) {
	want := errors.New("disk full")
	results := []Result{{
		Fixture: rule.Fixture{Name: "pass-alpha", Expect: rule.ExpectPass},
		Report:  lint.Report{Verdicts: map[string]int{"TestAlpha": 2}},
		Held:    true,
	}}
	err := Format(failingWriter{err: want}, Options{Rule: rule.Rule{Name: "demo"}, Votes: 2, Model: "sonnet"}, results, time.Second)
	if !errors.Is(err, want) {
		t.Fatalf("Format returned %v, want %v", err, want)
	}
}

func TestRun(t *testing.T) {
	root := t.TempDir()
	fixtures := []rule.Fixture{
		writeFixture(t, root, "pass-good", rule.ExpectPass, "TestGood"),
		{
			Name:     "pass-unreadable",
			TestFile: filepath.Join(root, "pass-unreadable", "scenario_test.go"),
			Expect:   rule.ExpectPass,
		},
		writeFixture(t, root, "fail-bad", rule.ExpectFail, "TestBad"),
	}
	opts := Options{
		Rule:  rule.Rule{Name: "one-reason-to-fail", Dir: root, Prompt: "A test has one reason to fail.", Granularity: rule.GranularityFunction},
		Votes: 3,
		Model: "sonnet",
	}
	ask := askFrom(map[string]reply{
		"TestGood": {
			names:    `{"names":["TestGood"]}`,
			verdicts: fmt.Sprintf(`{%q:{"satisfies":true,"reason":""}}`, keyOf(t, "TestGood")),
		},
		"TestBad": {
			names:    `{"names":["TestBad"]}`,
			verdicts: fmt.Sprintf(`{%q:{"satisfies":false,"reason":"names the unit"}}`, keyOf(t, "TestBad")),
		},
	})

	results := Run(context.Background(), ask, opts, fixtures)

	if len(results) != len(fixtures) {
		t.Fatalf("Run returned %d results, want %d", len(results), len(fixtures))
	}
	want := []struct {
		fixture  string
		held     bool
		wantErr  bool
		verdicts map[string]int
	}{
		{fixture: "pass-good", held: true, verdicts: map[string]int{"TestGood": 3}},
		{fixture: "pass-unreadable", wantErr: true},
		{fixture: "fail-bad", held: true, verdicts: map[string]int{"TestBad": 0}},
	}
	for i, tc := range want {
		t.Run(tc.fixture, func(t *testing.T) {
			got := results[i]
			if got.Fixture.Name != tc.fixture {
				t.Fatalf("result %d is fixture %q, want %q", i, got.Fixture.Name, tc.fixture)
			}
			if (got.Err != nil) != tc.wantErr {
				t.Fatalf("Err = %v, wantErr %t", got.Err, tc.wantErr)
			}
			if got.Held != tc.held {
				t.Fatalf("Held = %t, want %t", got.Held, tc.held)
			}
			if got.Report.Rule != opts.Rule.Name || got.Report.File != fixtures[i].TestFile || got.Report.Votes != opts.Votes {
				t.Fatalf("report = %+v, want rule %q file %q votes %d", got.Report, opts.Rule.Name, fixtures[i].TestFile, opts.Votes)
			}
			if !tc.wantErr && !maps.Equal(got.Report.Verdicts, tc.verdicts) {
				t.Fatalf("Verdicts = %v, want %v", got.Report.Verdicts, tc.verdicts)
			}
		})
	}
}

func resultOf(name string, held bool, err error) Result {
	return Result{Fixture: rule.Fixture{Name: name}, Held: held, Err: err}
}

type failingWriter struct{ err error }

func (f failingWriter) Write([]byte) (int, error) { return 0, f.err }

func writeFixture(t *testing.T, root, name string, expect rule.Expectation, testFunc string) rule.Fixture {
	t.Helper()
	dir := filepath.Join(root, name)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", dir, err)
	}
	path := filepath.Join(dir, "scenario_test.go")
	body := fmt.Sprintf("package scenario\n\nimport \"testing\"\n\nfunc %s(t *testing.T) {\n\tt.Log(\"scenario\")\n}\n", testFunc)
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
	return rule.Fixture{Name: name, TestFile: path, Expect: expect}
}

type reply struct {
	names    string
	verdicts string
}

func askFrom(replies map[string]reply) service.Ask {
	return func(_ context.Context, req service.Request) (json.RawMessage, error) {
		for marker, canned := range replies {
			if !strings.Contains(req.Prompt, marker) {
				continue
			}
			if isNamesCall(req) {
				return json.RawMessage(canned.names), nil
			}
			return json.RawMessage(canned.verdicts), nil
		}
		return nil, fmt.Errorf("no canned reply for prompt: %s", req.Prompt)
	}
}

func isNamesCall(req service.Request) bool {
	return strings.Contains(string(req.Schema), `"names"`)
}

// keyOf derives the property a unit answers under the same way the code under test
// does, so a canned verdict here tracks the key derivation rather than restating
// one spelling of it.
func keyOf(t *testing.T, name string) string {
	t.Helper()
	return lint.UnitsFor([]string{name})[0].Key
}
