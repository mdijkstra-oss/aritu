package lint

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"slices"
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
	testOnly := rule.Rule{Name: "named-for-behavior", Prompt: rulePrompt, Granularity: rule.GranularityFunction}
	withSource := rule.Rule{Name: "no-mocking", Prompt: rulePrompt, IncludeSource: true, Granularity: rule.GranularityFunction}

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

func TestApplyCollectsReasonsForUnitsThatFellShort(t *testing.T) {
	const bothPass = `{"results":[{"name":"TestParsesHost","reason":"","satisfies":true},{"name":"TestRejectsPort","reason":"","satisfies":true}]}`

	tests := []struct {
		name     string
		votes    int
		verdicts []cannedReply
		want     map[string][]string
	}{
		{
			name:     "a unanimous pass explains nothing",
			votes:    2,
			verdicts: repeat(2, ok(bothPass)),
			want:     nil,
		},
		{
			name:  "a split vote keeps only the dissenting run",
			votes: 2,
			verdicts: []cannedReply{
				ok(bothPass),
				ok(`{"results":[{"name":"TestParsesHost","reason":"","satisfies":true},{"name":"TestRejectsPort","reason":"names the unit, not the outcome","satisfies":false}]}`),
			},
			want: map[string][]string{"TestRejectsPort": {"names the unit, not the outcome"}},
		},
		{
			name:  "a unanimous rejection keeps one reason per dissenting run",
			votes: 2,
			verdicts: []cannedReply{
				ok(`{"results":[{"name":"TestParsesHost","reason":"first round on host","satisfies":false},{"name":"TestRejectsPort","reason":"first round on port","satisfies":false}]}`),
				ok(`{"results":[{"name":"TestParsesHost","reason":"second round on host","satisfies":false},{"name":"TestRejectsPort","reason":"second round on port","satisfies":false}]}`),
			},
			want: map[string][]string{
				"TestParsesHost":  {"first round on host", "second round on host"},
				"TestRejectsPort": {"first round on port", "second round on port"},
			},
		},
		{
			name:  "a blank reason is dropped rather than recorded as empty",
			votes: 1,
			verdicts: []cannedReply{
				ok(`{"results":[{"name":"TestParsesHost","reason":"   ","satisfies":false},{"name":"TestRejectsPort","reason":"port reason","satisfies":false}]}`),
			},
			want: map[string][]string{"TestRejectsPort": {"port reason"}},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			dir := writeFiles(t, nil)
			asker := &tableAsker{names: ok(bothNames), verdicts: tc.verdicts}
			opts := Options{
				Rule:  rule.Rule{Name: "named-for-behavior", Prompt: rulePrompt, Granularity: rule.GranularityFunction},
				File:  filepath.Join(dir, testFileName),
				Votes: tc.votes,
				Model: "sonnet",
			}

			report, err := Apply(context.Background(), asker.ask, opts)

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !maps.EqualFunc(report.Reasons, tc.want, sameReasons) {
				t.Errorf("reasons = %v, want %v", report.Reasons, tc.want)
			}
		})
	}
}

// sameReasons compares without regard to order: rounds run concurrently, so which
// dissenting run contributed which sentence is not a property worth pinning.
func sameReasons(got, want []string) bool {
	return slices.Equal(slices.Sorted(slices.Values(got)), slices.Sorted(slices.Values(want)))
}

func TestApplySkipsTheNamesCallAtFileGranularity(t *testing.T) {
	dir := writeFiles(t, nil)
	file := filepath.Join(dir, testFileName)
	asker := &tableAsker{
		names:    fails(errors.New("the names call must not be made at file granularity")),
		verdicts: repeat(2, ok(`{"results":[{"name":"FILE","reason":"","satisfies":true}]}`)),
	}
	opts := Options{
		Rule:  rule.Rule{Name: "shared-state", Prompt: rulePrompt, Granularity: rule.GranularityFile},
		File:  file,
		Votes: 2,
		Model: "sonnet",
	}
	asker.verdicts = repeat(2, ok(fmt.Sprintf(`{"results":[{"name":%q,"reason":"","satisfies":true}]}`, file)))

	report, err := Apply(context.Background(), asker.ask, opts)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !maps.Equal(report.Verdicts, map[string]int{file: 2}) {
		t.Errorf("verdicts = %v, want the file path keyed at 2", report.Verdicts)
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
	tests := []struct {
		name        string
		granularity rule.Granularity
		want        []string
	}{
		{
			name:        "function granularity asks for whole test functions",
			granularity: rule.GranularityFunction,
			want: []string{
				"top-level func", "begins with Test", "*testing.T",
				"helper functions", "table cases", "t.Run",
				"pkg/parser_test.go", "func TestRejectsPort(t *testing.T) {}",
			},
		},
		{
			name:        "test granularity asks for table rows and subtests as leaves",
			granularity: rule.GranularityTest,
			want: []string{
				"a case in a table", "t.Run", `"TestFunction (case name)"`,
				"#01", "built at run time",
				"pkg/parser_test.go", "func TestRejectsPort(t *testing.T) {}",
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			prompt := BuildNamesPrompt(tc.granularity, SourceFile{Path: "pkg/parser_test.go", Content: testFileSource})
			for _, want := range tc.want {
				if !strings.Contains(prompt, want) {
					t.Errorf("names prompt does not contain %q:\n%s", want, prompt)
				}
			}
		})
	}
}

func TestBuildNamesPromptPanicsAtFileGranularity(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Error("BuildNamesPrompt did not panic at file granularity, where no enumeration call should ever be made")
		}
	}()
	_ = BuildNamesPrompt(rule.GranularityFile, SourceFile{Path: "pkg/parser_test.go"})
}

func TestBuildVerdictPrompt(t *testing.T) {
	const base = "Judge the behavior a test pins down, never its syntax."
	files := []SourceFile{
		{Path: "pkg/parser_test.go", Content: testFileSource},
		{Path: "pkg/parser.go", Content: sourceFileSource},
	}
	units := []string{"TestParsesHost", "TestRejectsPort (empty input)"}
	prompt := BuildVerdictPrompt(base, "\n\n"+rulePrompt+"\n", files, units)

	t.Run("base prompt comes first", func(t *testing.T) {
		if !strings.HasPrefix(prompt, base) {
			t.Errorf("verdict prompt does not open with the base prompt:\n%s", prompt)
		}
	})

	t.Run("rule body follows the base prompt", func(t *testing.T) {
		if strings.Index(prompt, rulePrompt) < strings.Index(prompt, base) {
			t.Errorf("rule body precedes the base prompt:\n%s", prompt)
		}
	})

	tests := []struct {
		name string
		want string
	}{
		{"demands one entry per unit", "exactly one entry per unit"},
		{"lists the plain function unit", "- TestParsesHost"},
		{"lists the leaf unit", "- TestRejectsPort (empty input)"},
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

func TestBuildVerdictPromptOmitsAnEmptyBase(t *testing.T) {
	prompt := BuildVerdictPrompt("  \n ", rulePrompt, []SourceFile{{Path: "a_test.go"}}, []string{"TestA"})

	if !strings.HasPrefix(prompt, rulePrompt) {
		t.Errorf("an empty base should leave the rule body first:\n%s", prompt)
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
