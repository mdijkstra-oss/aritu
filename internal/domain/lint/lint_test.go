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
	"github.com/matthijn/aritu/internal/lib/service"
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

	noResults = `{}`
)

var (
	bothSatisfy = reply(answers(true, ""), answers(true, ""))
	portFails   = reply(answers(true, ""), answers(false, "names the unit"))
	noneSatisfy = reply(answers(false, "host reason"), answers(false, "port reason"))
	droppedName = reply(answers(true, ""))
	soloHost    = replyUnder([]string{"TestParsesHost"}, answers(true, ""))
	extraName   = replyUnder(
		[]string{"TestParsesHost", "TestRejectsPort", "TestGhost"},
		answers(true, ""), answers(true, ""), answers(true, ""),
	)
)

// reply renders a verdict for the file's two tests under the keys the schema
// actually names. Restating those keys as literals would pin every stub here to
// one spelling of the key derivation, so the stubs derive them the same way the
// code under test does: from the whole list, because a key carries the unit's
// position in it.
func reply(given ...verdictAnswer) string {
	return replyUnder([]string{"TestParsesHost", "TestRejectsPort"}, given...)
}

func replyUnder(names []string, given ...verdictAnswer) string {
	units := UnitsFor(names)
	rendered := map[string]verdictAnswer{}
	for at, answer := range given {
		rendered[units[at].Key] = answer
	}
	encoded, err := json.Marshal(rendered)
	if err != nil {
		panic(err)
	}
	return string(encoded)
}

func answers(satisfies bool, reason string) verdictAnswer {
	return verdictAnswer{Satisfies: satisfies, Reason: reason}
}

func TestApply(t *testing.T) {
	unreachable := errors.New("service: connection refused")
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
			name:         "a lone dissent passes",
			rule:         testOnly,
			votes:        4,
			names:        ok(bothNames),
			verdicts:     []cannedReply{ok(bothSatisfy), ok(bothSatisfy), ok(bothSatisfy), ok(portFails)},
			wantVerdicts: map[string]int{"TestParsesHost": 4, "TestRejectsPort": 3},
			wantExit:     ExitPass,
		},
		{
			name:         "a tie fails",
			rule:         testOnly,
			votes:        4,
			names:        ok(bothNames),
			verdicts:     []cannedReply{ok(bothSatisfy), ok(bothSatisfy), ok(portFails), ok(portFails)},
			wantVerdicts: map[string]int{"TestParsesHost": 4, "TestRejectsPort": 2},
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
			name:         "file with no tests passes vacuously",
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
			name:     "verdict naming a unit that was not listed errors",
			rule:     testOnly,
			votes:    4,
			names:    ok(bothNames),
			verdicts: repeat(4, ok(extraName)),
			wantErr:  "unexpected u03_test_ghost",
		},
		{
			name:     "verdict dropping a listed unit errors",
			rule:     testOnly,
			votes:    4,
			names:    ok(bothNames),
			verdicts: repeat(4, ok(droppedName)),
			wantErr:  "missing u02_test_rejects_port",
		},
		{
			name:         "duplicate names from the name call collapse into one unit",
			rule:         testOnly,
			votes:        4,
			names:        ok(sameName),
			verdicts:     repeat(4, ok(soloHost)),
			wantVerdicts: map[string]int{"TestParsesHost": 4},
			wantExit:     ExitPass,
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
			wantErr: "reading the unit names",
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
			name:     "a rule needing the source rejects a path no convention covers",
			files:    map[string]string{sourceFileName: sourceFileSource},
			fileName: sourceFileName,
			rule:     withSource,
			votes:    1,
			wantErr:  "matches no test file naming convention",
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
	bothPass := reply(answers(true, ""), answers(true, ""))

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
				ok(reply(answers(true, ""), answers(false, "names the unit, not the outcome"))),
			},
			want: map[string][]string{"TestRejectsPort": {"names the unit, not the outcome"}},
		},
		{
			name:  "a unanimous rejection keeps one reason per dissenting run",
			votes: 2,
			verdicts: []cannedReply{
				ok(reply(answers(false, "first round on host"), answers(false, "first round on port"))),
				ok(reply(answers(false, "second round on host"), answers(false, "second round on port"))),
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
				ok(reply(answers(false, "   "), answers(false, "port reason"))),
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
	key := UnitsAt(rule.GranularityFile, file, nil)[0].Key
	asker := &tableAsker{
		names:    fails(errors.New("the names call must not be made at file granularity")),
		verdicts: repeat(2, ok(fmt.Sprintf(`{%q:{"satisfies":true,"reason":"covered"}}`, key))),
	}
	opts := Options{
		Rule:  rule.Rule{Name: "shared-state", Prompt: rulePrompt, Granularity: rule.GranularityFile},
		File:  file,
		Votes: 2,
		Model: "sonnet",
	}
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
			name:   "a majority passes",
			report: Report{Votes: 4, Verdicts: map[string]int{"TestA": 4, "TestB": 3}},
			want:   ExitPass,
		},
		{
			name:   "a tie fails",
			report: Report{Votes: 4, Verdicts: map[string]int{"TestA": 4, "TestB": 2}},
			want:   ExitFail,
		},
		{
			name:   "unanimous rejection fails",
			report: Report{Votes: 4, Verdicts: map[string]int{"TestA": 0}},
			want:   ExitFail,
		},
		{
			name:   "no tests passes",
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

// TestBuildNamesPrompt asks each enumerating granularity for the same file. Each
// kind gets its own question: the splitter for functions asks for declarations,
// and the one for test cases asks for each leaf under its composite name.
func TestBuildNamesPrompt(t *testing.T) {
	tests := []struct {
		granularity rule.Granularity
		want        []string
	}{
		{
			granularity: rule.GranularityFunction,
			want: []string{
				"function or method", "declaration order",
				"pkg/parser_test.go", "func TestRejectsPort(t *testing.T) {}",
			},
		},
		{
			granularity: rule.GranularityTestCase,
			want: []string{
				"smallest thing this file's framework runs",
				"enclosing scope", `" > "`,
				"one row of a table of cases", "parametrised argument set",
				`"Name (case name)"`, "Parser > Address > ParsesHostBeforeColon",
				"#01", "built at run time",
				"helper functions", "setup and teardown hooks",
				"pkg/parser_test.go", "func TestRejectsPort(t *testing.T) {}",
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.granularity.String()+" granularity asks for its own kind of unit", func(t *testing.T) {
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
	files := []SourceFile{
		{Path: "pkg/parser_test.go", Content: testFileSource},
		{Path: "pkg/parser.go", Content: sourceFileSource},
	}
	units := UnitsFor([]string{"TestParsesHost", "TestRejectsPort (empty input)"})
	judged := rule.Rule{Name: "named-for-behavior", Granularity: rule.GranularityTestCase, Prompt: "\n\n" + rulePrompt + "\n"}
	prompt := BuildVerdictPrompt(judged, files, units)

	t.Run("the rule follows the shared guidance and precedes the units", func(t *testing.T) {
		shared := strings.Index(prompt, "<instructions>")
		rule := strings.Index(prompt, rulePrompt)
		listed := strings.Index(prompt, units[0].Name+"   ->   ")
		if !(shared < rule && rule < listed) {
			t.Errorf("layering is shared=%d rule=%d units=%d, want ascending:\n%s", shared, rule, listed, prompt)
		}
	})

	tests := []struct {
		name string
		want string
	}{
		{"heads the rule with the same title the rulebook gives it", "## Named for behavior"},
		{"names where the answer goes", "the key on the right is only where the answer goes"},
		{"judges the unit as written rather than the key", "as written on the left"},
		{"lists the plain unit against its key", "TestParsesHost   ->   " + units[0].Key},
		{"lists the leaf unit against its key", "TestRejectsPort (empty input)   ->   " + units[1].Key},
		{"names the test file", `<file path="pkg/parser_test.go">`},
		{"carries the test file contents", "func TestParsesHost(t *testing.T) {}"},
		{"names the source file", `<source path="pkg/parser.go">`},
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

func (a *tableAsker) ask(_ context.Context, req service.Request) (json.RawMessage, error) {
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
			name:  "a plain test is snake cased behind its position",
			names: []string{"TestParsesHost"},
			want:  []Unit{{Name: "TestParsesHost", Key: "u01_test_parses_host"}},
		},
		{
			name:  "punctuation and repeated spaces collapse to one separator",
			names: []string{"TestParse (rejects a 24:00 clock -- politely)"},
			want: []Unit{{
				Name: "TestParse (rejects a 24:00 clock -- politely)",
				Key:  "u01_test_parse_rejects_a_24_00_clock_politely",
			}},
		},
		{
			name:  "enclosing scopes read as words of one key",
			names: []string{"Parser > Address > ParsesHostBeforeColon"},
			want: []Unit{{
				Name: "Parser > Address > ParsesHostBeforeColon",
				Key:  "u01_parser_address_parses_host_before_colon",
			}},
		},
		{
			name:  "two cases that normalise alike are still told apart by their positions",
			names: []string{"TestParse (empty input)", "TestParse (empty  input)"},
			want: []Unit{
				{Name: "TestParse (empty input)", Key: "u01_test_parse_empty_input"},
				{Name: "TestParse (empty  input)", Key: "u02_test_parse_empty_input"},
			},
		},
		{
			name:  "a case with nothing to normalise still gets a key",
			names: []string{"TestParse (!!!)"},
			want:  []Unit{{Name: "TestParse (!!!)", Key: "u01_test_parse"}},
		},
		{
			name:  "a name that normalises to nothing is its position alone",
			names: []string{"!!!"},
			want:  []Unit{{Name: "!!!", Key: "u01"}},
		},
		{
			name:  "a path reads as one key with a word per segment",
			names: []string{"internal/parser/parser_test.go"},
			want: []Unit{{
				Name: "internal/parser/parser_test.go",
				Key:  "u01_internal_parser_parser_test_go",
			}},
		},
		{
			name:  "a name longer than the API allows keeps its tail",
			names: []string{"TestSelftestStillPrintsItsTable (when the model cannot be reached at all today)"},
			want: []Unit{{
				Name: "TestSelftestStillPrintsItsTable (when the model cannot be reached at all today)",
				Key:  "u01_ints_its_table_when_the_model_cannot_be_reached_at_all_today",
			}},
		},
		{
			name:  "the same function with different cases keeps them distinct",
			names: []string{"TestParse (accepts a port)", "TestParse (rejects a port)"},
			want: []Unit{
				{Name: "TestParse (accepts a port)", Key: "u01_test_parse_accepts_a_port"},
				{Name: "TestParse (rejects a port)", Key: "u02_test_parse_rejects_a_port"},
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

	wantKeys := []string{units[0].Key, units[1].Key}
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

// TestVerdictSchemaCarriesAdditionalPropertiesOnlyOnObjects guards a defect that
// costs a whole call and reports as the endpoint's complaint rather than as a bad
// schema: the format is sent strict, and additionalProperties beside a string or a
// boolean is rejected outright, so the target comes back as could-not-run.
func TestVerdictSchemaCarriesAdditionalPropertiesOnlyOnObjects(t *testing.T) {
	tests := []struct {
		name  string
		names []string
	}{
		{name: "one plain unit", names: []string{"TestParsesHost"}},
		{name: "a leaf carrying a case", names: []string{"TestParse (empty input)"}},
		{name: "a namespaced leaf", names: []string{"formatDate > pads days (2026-01-05)"}},
		{name: "several units", names: []string{"TestParsesHost", "TestParse (empty input)", "TestParse (blank input)"}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var node any
			if err := json.Unmarshal(VerdictSchemaFor(UnitsFor(tc.names)), &node); err != nil {
				t.Fatalf("schema is not valid JSON: %v", err)
			}

			for path, kind := range typedNodesIn(node, "#") {
				_, isConstrained := kind["additionalProperties"]
				if kind["type"] != "object" && isConstrained {
					t.Errorf("%s is a %v and still carries additionalProperties", path, kind["type"])
				}
				if kind["type"] == "object" && !isConstrained {
					t.Errorf("%s is an object that would accept a unit nobody enumerated", path)
				}
			}
		})
	}
}

// typedNodesIn walks every schema node carrying a type, keyed by its JSON pointer.
func typedNodesIn(node any, path string) map[string]map[string]any {
	found := map[string]map[string]any{}
	object, isObject := node.(map[string]any)
	if !isObject {
		return found
	}
	if _, isTyped := object["type"]; isTyped {
		found[path] = object
	}
	properties, hasProperties := object["properties"].(map[string]any)
	if !hasProperties {
		return found
	}
	for name, child := range properties {
		maps.Copy(found, typedNodesIn(child, path+"/properties/"+name))
	}
	return found
}

// TestUnitsAtFileGranularityAnswerUnderAKeyThatSurvivesBeingAToolParameter
// covers the shape that made a whole call unanswerable. A path cut to the ceiling is
// neither unique across files under one long directory nor legible, and a cut
// landing on a separator fails that derivation outright.
func TestUnitsAtFileGranularityAnswerUnderAKeyThatSurvivesBeingAToolParameter(t *testing.T) {
	tests := []struct {
		name string
		file string
		want string
	}{
		{
			name: "a short path stays readable in full",
			file: "internal/lib/vote/vote_test.go",
			want: "u01_internal_lib_vote_vote_test_go",
		},
		{
			name: "a path over the ceiling keeps its tail behind the position",
			file: "rules/no-gaps/fixtures/fail-go-insufficient-funds-never-reached/withdraw_test.go",
			want: "u01_es_fail_go_insufficient_funds_never_reached_withdraw_test_go",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			units := UnitsAt(rule.GranularityFile, tc.file, nil)

			if len(units) != 1 {
				t.Fatalf("UnitsAt() returned %d units, want the file as one", len(units))
			}
			if units[0].Name != tc.file {
				t.Errorf("unit name = %q, want the path the report prints", units[0].Name)
			}
			if units[0].Key != tc.want {
				t.Errorf("key = %q, want %q", units[0].Key, tc.want)
			}
			if len(units[0].Key) > maxKeyLength {
				t.Errorf("key %q is %d chars, over the ceiling of %d", units[0].Key, len(units[0].Key), maxKeyLength)
			}
		})
	}
}

// TestKeysStayUniqueWhereTruncationWouldNot is the reason the position prefix
// beats a bare cut: two files under one long directory reduce to the same string,
// and a schema cannot carry the same property twice.
func TestKeysStayUniqueWhereTruncationWouldNot(t *testing.T) {
	const dir = "rules/no-redundancy/fixtures/pass-go-boundary-pair-at-and-past-limit/"

	units := UnitsFor([]string{dir + "retention_test.go", dir + "expiry_test.go"})

	if units[0].Key == units[1].Key {
		t.Errorf("both files answer under %q, so one verdict would overwrite the other", units[0].Key)
	}
	for _, unit := range units {
		if len(unit.Key) > maxKeyLength {
			t.Errorf("key %q is %d chars, over the ceiling of %d", unit.Key, len(unit.Key), maxKeyLength)
		}
	}
}

// TestKeysNeverEndOnASeparator pins the key shape a verdict is answered under.
// A name at the length ceiling that ends on a separator once made a whole call
// unanswerable, and the key is still what every report is keyed by.
func TestKeysNeverEndOnASeparator(t *testing.T) {
	tests := []struct {
		name  string
		unit  string
		wantK string
	}{
		{
			name:  "a path over the ceiling is cut without a separator surviving the cut",
			unit:  "rules/no-gaps/fixtures/fail-go-insufficient-funds-never-reached/withdraw_test.go",
			wantK: "u01_es_fail_go_insufficient_funds_never_reached_withdraw_test_go",
		},
		{
			name:  "a case name ending in punctuation keeps nothing trailing",
			unit:  "TestParse (rejects blank input --)",
			wantK: "u01_test_parse_rejects_blank_input",
		},
		{
			name:  "a case that normalises to nothing leaves no separator behind it",
			unit:  "TestParse (!!!)",
			wantK: "u01_test_parse",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := UnitsFor([]string{tc.unit})[0].Key

			if got != tc.wantK {
				t.Errorf("key = %q, want %q", got, tc.wantK)
			}
			if strings.HasSuffix(got, "_") || strings.HasSuffix(got, "-") || strings.HasSuffix(got, ".") {
				t.Errorf("key %q ends on a separator", got)
			}
		})
	}
}
