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

	bothSatisfy = `{"TestParsesHost":{"satisfies":true,"reason":""},"TestRejectsPort":{"satisfies":true,"reason":""}}`
	portFails   = `{"TestParsesHost":{"satisfies":true,"reason":""},"TestRejectsPort":{"satisfies":false,"reason":"names the unit"}}`
	noneSatisfy = `{"TestParsesHost":{"satisfies":false,"reason":"host reason"},"TestRejectsPort":{"satisfies":false,"reason":"port reason"}}`
	extraName   = `{"TestParsesHost":{"satisfies":true,"reason":""},"TestRejectsPort":{"satisfies":true,"reason":""},"TestGhost":{"satisfies":true,"reason":""}}`
	droppedName = `{"TestParsesHost":{"satisfies":true,"reason":""}}`
	noResults   = `{}`
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
			verdicts: repeat(1, ok(`{"TestParsesHost":`)),
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
	const bothPass = `{"TestParsesHost":{"satisfies":true,"reason":""},"TestRejectsPort":{"satisfies":true,"reason":""}}`

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
				ok(`{"TestParsesHost":{"satisfies":true,"reason":""},"TestRejectsPort":{"satisfies":false,"reason":"names the unit, not the outcome"}}`),
			},
			want: map[string][]string{"TestRejectsPort": {"names the unit, not the outcome"}},
		},
		{
			name:  "a unanimous rejection keeps one reason per dissenting run",
			votes: 2,
			verdicts: []cannedReply{
				ok(`{"TestParsesHost":{"satisfies":false,"reason":"first round on host"},"TestRejectsPort":{"satisfies":false,"reason":"first round on port"}}`),
				ok(`{"TestParsesHost":{"satisfies":false,"reason":"second round on host"},"TestRejectsPort":{"satisfies":false,"reason":"second round on port"}}`),
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
				ok(`{"TestParsesHost":{"satisfies":false,"reason":"   "},"TestRejectsPort":{"satisfies":false,"reason":"port reason"}}`),
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
		verdicts: repeat(2, ok(`{"FILE":{"satisfies":true,"reason":""}}`)),
	}
	opts := Options{
		Rule:  rule.Rule{Name: "shared-state", Prompt: rulePrompt, Granularity: rule.GranularityFile},
		File:  file,
		Votes: 2,
		Model: "sonnet",
	}
	asker.verdicts = repeat(2, ok(fmt.Sprintf(`{%q:{"satisfies":true,"reason":""}}`, UnitsFor([]string{file})[0].Key)))

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
	units := UnitsFor([]string{"TestParsesHost", "TestRejectsPort (empty input)"})
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
		{"names the key to answer under", "the key to answer under"},
		{"judges the unit as written rather than the key", "as written on the left"},
		{"lists the plain function unit against itself", "- TestParsesHost   ->   TestParsesHost"},
		{"lists the leaf unit against its key", "- TestRejectsPort (empty input)   ->   TestRejectsPort.empty_input"},
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
	prompt := BuildVerdictPrompt("  \n ", rulePrompt, []SourceFile{{Path: "a_test.go"}}, UnitsFor([]string{"TestA"}))

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

func TestUnitsFor(t *testing.T) {
	tests := []struct {
		name  string
		names []string
		want  []Unit
	}{
		{
			name:  "a plain test function answers under its own name",
			names: []string{"TestParsesHost"},
			want:  []Unit{{Name: "TestParsesHost", Key: "TestParsesHost"}},
		},
		{
			name:  "a case name is snake cased behind the function name",
			names: []string{"TestParseConfig (extracts host before colon)"},
			want:  []Unit{{Name: "TestParseConfig (extracts host before colon)", Key: "TestParseConfig.extracts_host_before_colon"}},
		},
		{
			name:  "punctuation and repeated spaces collapse to one separator",
			names: []string{"TestParse (rejects a 24:00 clock -- politely)"},
			want:  []Unit{{Name: "TestParse (rejects a 24:00 clock -- politely)", Key: "TestParse.rejects_a_24_00_clock_politely"}},
		},
		{
			name:  "two cases that normalise alike are kept apart",
			names: []string{"TestParse (empty input)", "TestParse (empty  input)"},
			want: []Unit{
				{Name: "TestParse (empty input)", Key: "TestParse.empty_input"},
				{Name: "TestParse (empty  input)", Key: "TestParse.empty_input-01"},
			},
		},
		{
			name:  "a case with nothing to normalise still gets a key",
			names: []string{"TestParse (!!!)"},
			want:  []Unit{{Name: "TestParse (!!!)", Key: "TestParse.case"}},
		},
		{
			name:  "a file path is sanitised into the key character set",
			names: []string{"internal/parser/parser_test.go"},
			want:  []Unit{{Name: "internal/parser/parser_test.go", Key: "internal_parser_parser_test.go"}},
		},
		{
			name:  "a key longer than the API allows is cut to the ceiling",
			names: []string{"TestSelftestStillPrintsItsTable (when the model cannot be reached at all today)"},
			want: []Unit{{
				Name: "TestSelftestStillPrintsItsTable (when the model cannot be reached at all today)",
				Key:  "TestSelftestStillPrintsItsTable.when_the_model_cannot_be_reached",
			}},
		},
		{
			name:  "the same function with different cases keeps them distinct",
			names: []string{"TestParse (accepts a port)", "TestParse (rejects a port)"},
			want: []Unit{
				{Name: "TestParse (accepts a port)", Key: "TestParse.accepts_a_port"},
				{Name: "TestParse (rejects a port)", Key: "TestParse.rejects_a_port"},
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := UnitsFor(tc.names)
			if !slices.Equal(got, tc.want) {
				t.Errorf("UnitsFor(%q) = %+v, want %+v", tc.names, got, tc.want)
			}
			for _, unit := range got {
				if len(unit.Key) > maxKeyLength {
					t.Errorf("key %q is %d chars, over the API ceiling of %d", unit.Key, len(unit.Key), maxKeyLength)
				}
			}
		})
	}
}

func TestVerdictSchemaForNamesEveryKeyAndForbidsAnyOther(t *testing.T) {
	units := UnitsFor([]string{"TestParsesHost", "TestParse (empty input)"})

	raw := VerdictSchemaFor(units)

	var schema struct {
		Type                 string                     `json:"type"`
		Properties           map[string]json.RawMessage `json:"properties"`
		Required             []string                   `json:"required"`
		AdditionalProperties bool                       `json:"additionalProperties"`
	}
	if err := json.Unmarshal(raw, &schema); err != nil {
		t.Fatalf("schema is not valid JSON: %v", err)
	}

	wantKeys := []string{"TestParsesHost", "TestParse.empty_input"}
	if !slices.Equal(slices.Sorted(maps.Keys(schema.Properties)), slices.Sorted(slices.Values(wantKeys))) {
		t.Errorf("properties = %v, want %v", slices.Sorted(maps.Keys(schema.Properties)), wantKeys)
	}
	if !slices.Equal(slices.Sorted(slices.Values(schema.Required)), slices.Sorted(slices.Values(wantKeys))) {
		t.Errorf("required = %v, want every key", schema.Required)
	}
	if schema.AdditionalProperties {
		t.Error("additionalProperties is true, so the model could invent a unit the enumeration never listed")
	}
}

func TestVerdictSchemaForEscapesAKeyRatherThanBreakingTheJSON(t *testing.T) {
	raw := VerdictSchemaFor([]Unit{{Name: `Test"quoted`, Key: `Test_quoted`}})
	if !json.Valid(raw) {
		t.Fatalf("schema is not valid JSON: %s", raw)
	}
}
