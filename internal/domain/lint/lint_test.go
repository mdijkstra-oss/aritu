package lint

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/matthijn/aritu/internal/domain/rule"
	"github.com/matthijn/aritu/internal/lib/claudecli"
)

const (
	testFileName   = "parser_test.go"
	sourceFileName = "parser.go"

	testFileSource = `package parser

import "testing"

func TestParsesHost(t *testing.T) {}

func TestRejectsPort(t *testing.T) {}
`
	sourceFileSource = `package parser

func Parse(raw string) error { return nil }
`

	rulePrompt = "A test is named for the behavior it protects, specifically."

	bothNames = `{"names":["TestParsesHost","TestRejectsPort"]}`
	noNames   = `{"names":[]}`
	sameName  = `{"names":["TestParsesHost","TestParsesHost"]}`

	bothSatisfy = `{"results":[{"name":"TestParsesHost","satisfies":true},{"name":"TestRejectsPort","satisfies":true}]}`
	portFails   = `{"results":[{"name":"TestParsesHost","satisfies":true},{"name":"TestRejectsPort","satisfies":false}]}`
	noneSatisfy = `{"results":[{"name":"TestParsesHost","satisfies":false},{"name":"TestRejectsPort","satisfies":false}]}`
	extraName   = `{"results":[{"name":"TestParsesHost","satisfies":true},{"name":"TestRejectsPort","satisfies":true},{"name":"TestGhost","satisfies":true}]}`
	droppedName = `{"results":[{"name":"TestParsesHost","satisfies":true}]}`
	repeatedRow = `{"results":[{"name":"TestParsesHost","satisfies":true},{"name":"TestParsesHost","satisfies":false},{"name":"TestRejectsPort","satisfies":true}]}`
	noResults   = `{"results":[]}`
)

func TestApply(t *testing.T) {
	unreachable := errors.New("claude: connection refused")
	testOnly := rule.Rule{Name: "named-for-behavior", Prompt: rulePrompt}
	withSource := rule.Rule{Name: "no-mocking", Prompt: rulePrompt, IncludeSource: true}

	tests := []struct {
		name               string
		files              map[string]string
		fileName           string
		rule               rule.Rule
		votes              int
		names              cannedReply
		verdicts           []cannedReply
		wantVerdicts       map[string]int
		wantExit           Exit
		wantPromptContains []string
		wantErr            string
	}{
		{
			name:         "full agreement passes",
			rule:         testOnly,
			votes:        4,
			names:        ok(bothNames),
			verdicts:     repeat(4, ok(bothSatisfy)),
			wantVerdicts: map[string]int{"TestParsesHost": 4, "TestRejectsPort": 4},
			wantExit:     ExitPass,
		},
		{
			name:         "split vote fails",
			rule:         testOnly,
			votes:        4,
			names:        ok(bothNames),
			verdicts:     []cannedReply{ok(bothSatisfy), ok(bothSatisfy), ok(bothSatisfy), ok(portFails)},
			wantVerdicts: map[string]int{"TestParsesHost": 4, "TestRejectsPort": 3},
			wantExit:     ExitFail,
		},
		{
			name:         "unanimous rejection fails",
			rule:         testOnly,
			votes:        4,
			names:        ok(bothNames),
			verdicts:     repeat(4, ok(noneSatisfy)),
			wantVerdicts: map[string]int{"TestParsesHost": 0, "TestRejectsPort": 0},
			wantExit:     ExitFail,
		},
		{
			name:         "file without test functions passes vacuously",
			rule:         testOnly,
			votes:        4,
			names:        ok(noNames),
			verdicts:     repeat(4, ok(noResults)),
			wantVerdicts: map[string]int{},
			wantExit:     ExitPass,
		},
		{
			name:     "source file reaches the model",
			files:    map[string]string{testFileName: testFileSource, sourceFileName: sourceFileSource},
			rule:     withSource,
			votes:    1,
			names:    ok(bothNames),
			verdicts: repeat(1, ok(bothSatisfy)),
			wantVerdicts: map[string]int{
				"TestParsesHost": 1, "TestRejectsPort": 1,
			},
			wantExit:           ExitPass,
			wantPromptContains: []string{rulePrompt, testFileName, sourceFileName, "func Parse(raw string) error"},
		},
		{
			name:     "verdict naming a function that was not listed errors",
			rule:     testOnly,
			votes:    4,
			names:    ok(bothNames),
			verdicts: repeat(4, ok(extraName)),
			wantErr:  "unexpected TestGhost",
		},
		{
			name:     "verdict dropping a listed function errors",
			rule:     testOnly,
			votes:    4,
			names:    ok(bothNames),
			verdicts: repeat(4, ok(droppedName)),
			wantErr:  "missing TestRejectsPort",
		},
		{
			name:     "verdict given twice for one function errors",
			rule:     testOnly,
			votes:    1,
			names:    ok(bothNames),
			verdicts: repeat(1, ok(repeatedRow)),
			wantErr:  `"TestParsesHost" given twice`,
		},
		{
			name:    "duplicate names from the name call error",
			rule:    testOnly,
			votes:   4,
			names:   ok(sameName),
			wantErr: `"TestParsesHost" listed more than once`,
		},
		{
			name:    "votes below one errors",
			rule:    testOnly,
			votes:   0,
			names:   ok(bothNames),
			wantErr: "votes must be at least 1",
		},
		{
			name:    "missing source file errors when the rule needs it",
			rule:    withSource,
			votes:   4,
			names:   ok(bothNames),
			wantErr: sourceFileName,
		},
		{
			name:    "name call failure errors",
			rule:    testOnly,
			votes:   4,
			names:   fails(unreachable),
			wantErr: "connection refused",
		},
		{
			name:     "verdict call failure errors",
			rule:     testOnly,
			votes:    1,
			names:    ok(bothNames),
			verdicts: []cannedReply{fails(unreachable)},
			wantErr:  "connection refused",
		},
		{
			name:    "unparseable name reply errors",
			rule:    testOnly,
			votes:   4,
			names:   ok(`{"names":`),
			wantErr: "reading test function names",
		},
		{
			name:     "unparseable verdict reply errors",
			rule:     testOnly,
			votes:    1,
			names:    ok(bothNames),
			verdicts: repeat(1, ok(`{"results":`)),
			wantErr:  "reading verdicts",
		},
		{
			name:    "an unreadable test file errors before the model is asked",
			files:   map[string]string{},
			rule:    testOnly,
			votes:   1,
			wantErr: "no such file",
		},
		{
			name:     "a rule needing the source rejects a path that is not a test file",
			files:    map[string]string{sourceFileName: sourceFileSource},
			fileName: sourceFileName,
			rule:     withSource,
			votes:    1,
			wantErr:  "is not a Go test file",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			dir := writeFiles(t, tc.files)
			name := tc.fileName
			if name == "" {
				name = testFileName
			}
			file := filepath.Join(dir, name)
			asker := &tableAsker{names: tc.names, verdicts: tc.verdicts}
			opts := Options{Rule: tc.rule, File: file, Votes: tc.votes, Model: "sonnet"}

			report, err := Apply(context.Background(), asker.ask, opts)

			if report.Rule != tc.rule.Name || report.File != file || report.Votes != tc.votes {
				t.Errorf("report identity = %q/%q/%d, want %q/%q/%d",
					report.Rule, report.File, report.Votes, tc.rule.Name, file, tc.votes)
			}

			if tc.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
					t.Fatalf("error = %v, want one containing %q", err, tc.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !maps.Equal(report.Verdicts, tc.wantVerdicts) {
				t.Errorf("verdicts = %v, want %v", report.Verdicts, tc.wantVerdicts)
			}
			if got := ExitFor(report); got != tc.wantExit {
				t.Errorf("ExitFor = %d, want %d", got, tc.wantExit)
			}
			for _, want := range tc.wantPromptContains {
				if !strings.Contains(asker.firstVerdictPrompt(t), want) {
					t.Errorf("verdict prompt does not contain %q", want)
				}
			}
		})
	}
}

func TestExitFor(t *testing.T) {
	tests := []struct {
		name   string
		report Report
		want   Exit
	}{
		{
			name:   "full agreement passes",
			report: Report{Votes: 4, Verdicts: map[string]int{"TestA": 4, "TestB": 4}},
			want:   ExitPass,
		},
		{
			name:   "one vote short fails",
			report: Report{Votes: 4, Verdicts: map[string]int{"TestA": 4, "TestB": 3}},
			want:   ExitFail,
		},
		{
			name:   "unanimous rejection fails",
			report: Report{Votes: 4, Verdicts: map[string]int{"TestA": 0}},
			want:   ExitFail,
		},
		{
			name:   "no test functions passes",
			report: Report{Votes: 4, Verdicts: map[string]int{}},
			want:   ExitPass,
		},
		{
			name:   "single vote counts as full agreement",
			report: Report{Votes: 1, Verdicts: map[string]int{"TestA": 1}},
			want:   ExitPass,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := ExitFor(tc.report); got != tc.want {
				t.Errorf("ExitFor = %d, want %d", got, tc.want)
			}
		})
	}
}

func TestBuildNamesPrompt(t *testing.T) {
	prompt := BuildNamesPrompt(SourceFile{Path: "pkg/parser_test.go", Content: testFileSource})

	tests := []struct {
		name string
		want string
	}{
		{"requires a top-level func", "top-level func"},
		{"requires the Test prefix", "begins with Test"},
		{"requires the testing.T parameter", "*testing.T"},
		{"excludes helpers", "helper functions"},
		{"excludes table cases", "table cases"},
		{"excludes subtest closures", "t.Run"},
		{"names the file", "pkg/parser_test.go"},
		{"carries the file contents", "func TestRejectsPort(t *testing.T) {}"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if !strings.Contains(prompt, tc.want) {
				t.Errorf("names prompt does not contain %q:\n%s", tc.want, prompt)
			}
		})
	}
}

func TestBuildVerdictPrompt(t *testing.T) {
	files := []SourceFile{
		{Path: "pkg/parser_test.go", Content: testFileSource},
		{Path: "pkg/parser.go", Content: sourceFileSource},
	}
	prompt := BuildVerdictPrompt("\n\n"+rulePrompt+"\n", files)

	t.Run("rule body comes first", func(t *testing.T) {
		if !strings.HasPrefix(prompt, rulePrompt) {
			t.Errorf("verdict prompt does not open with the rule body:\n%s", prompt)
		}
	})

	tests := []struct {
		name string
		want string
	}{
		{"demands one entry per test function", "exactly one entry"},
		{"allows for table-driven tests", "table-driven test"},
		{"allows for subtests", "subtests under t.Run"},
		{"allows for functional tests", "functional test"},
		{"judges behavior over syntax", "not the syntax"},
		{"names the test file", "pkg/parser_test.go"},
		{"carries the test file contents", "func TestParsesHost(t *testing.T) {}"},
		{"names the source file", "pkg/parser.go"},
		{"carries the source file contents", "func Parse(raw string) error"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if !strings.Contains(prompt, tc.want) {
				t.Errorf("verdict prompt does not contain %q:\n%s", tc.want, prompt)
			}
		})
	}
}

type cannedReply struct {
	body string
	err  error
}

func ok(body string) cannedReply { return cannedReply{body: body} }

func fails(err error) cannedReply { return cannedReply{err: err} }

func repeat(n int, reply cannedReply) []cannedReply {
	replies := make([]cannedReply, n)
	for i := range replies {
		replies[i] = reply
	}
	return replies
}

type tableAsker struct {
	names          cannedReply
	verdicts       []cannedReply
	mu             sync.Mutex
	used           int
	verdictPrompts []string
}

func (a *tableAsker) ask(_ context.Context, req claudecli.Request) (json.RawMessage, error) {
	if string(req.Schema) == NamesSchema {
		return json.RawMessage(a.names.body), a.names.err
	}

	a.mu.Lock()
	defer a.mu.Unlock()
	a.verdictPrompts = append(a.verdictPrompts, req.Prompt)
	if a.used >= len(a.verdicts) {
		return nil, fmt.Errorf("no canned reply for verdict round %d", a.used+1)
	}
	reply := a.verdicts[a.used]
	a.used++
	return json.RawMessage(reply.body), reply.err
}

func (a *tableAsker) firstVerdictPrompt(t *testing.T) string {
	t.Helper()
	a.mu.Lock()
	defer a.mu.Unlock()
	if len(a.verdictPrompts) == 0 {
		t.Fatal("no verdict prompt was built")
	}
	return a.verdictPrompts[0]
}

func writeFiles(t *testing.T, files map[string]string) string {
	t.Helper()
	dir := t.TempDir()
	wanted := files
	if wanted == nil {
		wanted = map[string]string{testFileName: testFileSource}
	}
	for name, content := range wanted {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o600); err != nil {
			t.Fatalf("writing %s: %v", name, err)
		}
	}
	return dir
}
