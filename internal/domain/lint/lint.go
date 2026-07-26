package lint

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/matthijn/aritu/internal/domain/rule"
	"github.com/matthijn/aritu/internal/lib/claudecli"
	"github.com/matthijn/aritu/internal/lib/vote"
)

// Exit is a process exit status. A split vote is a rule failure, never a
// could-not-run: routing it to ExitError would invite a commit hook to treat an
// unsure model as a tooling problem and skip past exactly the test this tool
// exists to catch.
type Exit int

// SourceFile is a file's path and contents as handed to the model.
type SourceFile struct {
	Path    string
	Content string
}

// Report is the tool's output. Verdicts maps each judged unit to how many of
// Votes runs judged it to satisfy the rule; only a count equal to Votes passes.
// Reasons carries one explanation per dissenting run, for units that fell short.
type Report struct {
	Rule     string              `json:"rule"`
	File     string              `json:"file"`
	Votes    int                 `json:"votes"`
	Verdicts map[string]int      `json:"verdicts"`
	Reasons  map[string][]string `json:"reasons,omitempty"`
	Error    string              `json:"error,omitempty"`
}

// Options configures one Apply run.
type Options struct {
	Rule   rule.Rule
	Base   string
	File   string
	Votes  int
	Model  string
	Effort string
}

const (
	ExitPass  Exit = 0
	ExitFail  Exit = 1
	ExitError Exit = 2
)

// NamesSchema constrains the test-name call. Keys are fixed because dynamic-key
// schemas exhaust the CLI's structured-output retries.
const NamesSchema = `{"type":"object","properties":{"names":{"type":"array","items":{"type":"string"}}},"required":["names"],"additionalProperties":false}`

// VerdictSchema constrains the per-unit verdict call.
const VerdictSchema = `{"type":"object","properties":{"results":{"type":"array","items":{"type":"object","properties":{"name":{"type":"string"},"reason":{"type":"string"},"satisfies":{"type":"boolean"}},"required":["name","reason","satisfies"],"additionalProperties":false}}},"required":["results"],"additionalProperties":false}`

// Apply votes on one file against one rule. The returned Report carries Rule,
// File and Votes even when the error is non-nil, so the caller can always emit
// output before exiting.
func Apply(ctx context.Context, ask claudecli.Ask, opts Options) (Report, error) {
	report := Report{Rule: opts.Rule.Name, File: opts.File, Votes: opts.Votes}

	if opts.Votes < 1 {
		return report, fmt.Errorf("votes must be at least 1, got %d", opts.Votes)
	}

	files, err := readFiles(opts.Rule, opts.File)
	if err != nil {
		return report, err
	}

	units, err := listUnits(ctx, ask, opts, files[0])
	if err != nil {
		return report, err
	}

	judge := func(ctx context.Context, _ int) (round, error) {
		return askVerdicts(ctx, ask, opts, files, units)
	}
	rounds, err := vote.Collect(ctx, opts.Votes, judge)
	if err != nil {
		return report, err
	}

	report.Verdicts = vote.Tally(verdictsOf(rounds))
	report.Reasons = collectReasons(rounds, report.Verdicts, opts.Votes)
	return report, nil
}

// ExitFor derives the exit status of a completed report.
func ExitFor(r Report) Exit {
	if vote.IsUnanimous(r.Verdicts, r.Votes) {
		return ExitPass
	}
	return ExitFail
}

// BuildNamesPrompt asks the model to enumerate the units a rule judges. It is
// never called at file granularity, where the unit is the path and no model is
// needed to know it.
func BuildNamesPrompt(granularity rule.Granularity, test SourceFile) string {
	build, isKnown := namesPrompts[granularity]
	if !isKnown {
		panic(fmt.Sprintf("no names prompt for granularity: %s", granularity))
	}
	return build(test)
}

// BuildVerdictPrompt frames the shared base prompt and one rule's criterion
// around the files under judgement and the exact units to judge. The units are
// listed rather than left to be re-derived: two independent enumerations of
// twenty-five table rows will phrase one of them differently sooner or later,
// and every such disagreement would surface as a could-not-run.
func BuildVerdictPrompt(base, rulePrompt string, files []SourceFile, units []string) string {
	var b strings.Builder
	if trimmed := strings.TrimSpace(base); trimmed != "" {
		b.WriteString(trimmed)
		b.WriteString("\n\n---\n\n")
	}
	b.WriteString(strings.TrimSpace(rulePrompt))
	b.WriteString("\n\n---\n\n")
	b.WriteString("Judge exactly these units against the rule above:\n")
	for _, unit := range units {
		b.WriteString("- " + unit + "\n")
	}
	b.WriteString("\nReturn exactly one entry per unit listed, no more and no fewer, naming each unit exactly as written above.\n\n")
	for _, f := range files {
		b.WriteString(formatFileBlock(f))
		b.WriteString("\n")
	}
	return b.String()
}

var namesPrompts = map[rule.Granularity]func(SourceFile) string{
	rule.GranularityFunction: buildFunctionNamesPrompt,
	rule.GranularityTest:     buildTestNamesPrompt,
}

func buildFunctionNamesPrompt(test SourceFile) string {
	var b strings.Builder
	b.WriteString("List every test function declared in the Go test file below.\n\n")
	b.WriteString("A test function is a top-level func whose name begins with Test and which takes a single *testing.T parameter.\n\n")
	b.WriteString("Do not list any of the following:\n")
	b.WriteString("- helper functions, including helpers that take *testing.T\n")
	b.WriteString("- struct types or literals holding table cases, and the names of those cases\n")
	b.WriteString("- subtest closures passed to t.Run, however they are named\n")
	b.WriteString("- benchmarks, fuzz targets and examples\n\n")
	b.WriteString("Report each name exactly as declared, in declaration order.\n\n")
	b.WriteString(formatFileBlock(test))
	return b.String()
}

func buildTestNamesPrompt(test SourceFile) string {
	var b strings.Builder
	b.WriteString("List every test unit in the Go test file below.\n\n")
	b.WriteString("A test unit is one leaf that can fail on its own and be named:\n")
	b.WriteString("- a case in a table, named by its name field or by its map key\n")
	b.WriteString("- a subtest declared with t.Run\n")
	b.WriteString("- a top-level func Test* that declares neither, which is one unit by itself\n\n")
	b.WriteString("Write a leaf inside a test function as \"TestFunction (case name)\", taking the case name exactly as it appears in the source. Write a test function that declares no cases as just \"TestFunction\".\n\n")
	b.WriteString("When two cases in one function share a name, disambiguate the second and later ones the way Go does, by appending #01, #02 and so on.\n\n")
	b.WriteString("Do not list any of the following:\n")
	b.WriteString("- helper functions, including helpers that take *testing.T\n")
	b.WriteString("- the struct type or the field names of a table, only its cases\n")
	b.WriteString("- benchmarks, fuzz targets and examples\n")
	b.WriteString("- cases whose name is built at run time rather than written in the source; when a function's cases cannot be named from the source, list that function itself as one unit\n\n")
	b.WriteString("Report units in declaration order.\n\n")
	b.WriteString(formatFileBlock(test))
	return b.String()
}

type namesReply struct {
	Names []string `json:"names"`
}

type verdictReply struct {
	Results []verdictEntry `json:"results"`
}

type verdictEntry struct {
	Name      string `json:"name"`
	Reason    string `json:"reason"`
	Satisfies bool   `json:"satisfies"`
}

// round is one run's answer: a verdict per unit, and the model's one-line
// justification for each.
type round struct {
	verdicts map[string]bool
	reasons  map[string]string
}

func readFiles(r rule.Rule, testPath string) ([]SourceFile, error) {
	test, err := readSourceFile(testPath)
	if err != nil {
		return nil, err
	}
	if !r.IncludeSource {
		return []SourceFile{test}, nil
	}

	sourcePath, isTestFile := rule.SourcePathFor(testPath)
	if !isTestFile {
		return nil, fmt.Errorf("rule %s needs the file under test but %s is not a Go test file", r.Name, testPath)
	}
	source, err := readSourceFile(sourcePath)
	if err != nil {
		return nil, fmt.Errorf("rule %s needs the file under test: %w", r.Name, err)
	}
	return []SourceFile{test, source}, nil
}

func readSourceFile(path string) (SourceFile, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return SourceFile{}, err
	}
	return SourceFile{Path: path, Content: string(content)}, nil
}

func askNames(ctx context.Context, ask claudecli.Ask, opts Options, test SourceFile) ([]string, error) {
	raw, err := ask(ctx, claudecli.Request{
		Prompt: BuildNamesPrompt(opts.Rule.Granularity, test),
		Model:  opts.Model,
		Effort: opts.Effort,
		Schema: json.RawMessage(NamesSchema),
	})
	if err != nil {
		return nil, fmt.Errorf("listing test functions in %s: %w", test.Path, err)
	}

	var reply namesReply
	if err := json.Unmarshal(raw, &reply); err != nil {
		return nil, fmt.Errorf("reading test function names for %s: %w", test.Path, err)
	}
	if duplicate, hasDuplicate := findDuplicate(reply.Names); hasDuplicate {
		return nil, fmt.Errorf("test unit %q listed more than once in %s", duplicate, test.Path)
	}
	return reply.Names, nil
}

// listUnits enumerates what this rule judges. At file granularity there is
// nothing to enumerate, so the call is skipped: the unit is the path, which
// costs no tokens to know and cannot be disagreed with.
func listUnits(ctx context.Context, ask claudecli.Ask, opts Options, test SourceFile) ([]string, error) {
	if opts.Rule.Granularity == rule.GranularityFile {
		return []string{opts.File}, nil
	}
	return askNames(ctx, ask, opts, test)
}

func askVerdicts(ctx context.Context, ask claudecli.Ask, opts Options, files []SourceFile, units []string) (round, error) {
	raw, err := ask(ctx, claudecli.Request{
		Prompt: BuildVerdictPrompt(opts.Base, opts.Rule.Prompt, files, units),
		Model:  opts.Model,
		Effort: opts.Effort,
		Schema: json.RawMessage(VerdictSchema),
	})
	if err != nil {
		return round{}, fmt.Errorf("judging %s against rule %s: %w", opts.File, opts.Rule.Name, err)
	}

	var reply verdictReply
	if err := json.Unmarshal(raw, &reply); err != nil {
		return round{}, fmt.Errorf("reading verdicts for %s: %w", opts.File, err)
	}

	judged := round{
		verdicts: make(map[string]bool, len(reply.Results)),
		reasons:  make(map[string]string, len(reply.Results)),
	}
	for _, entry := range reply.Results {
		if _, isRepeat := judged.verdicts[entry.Name]; isRepeat {
			return round{}, fmt.Errorf("verdict for test unit %q given twice in %s", entry.Name, opts.File)
		}
		judged.verdicts[entry.Name] = entry.Satisfies
		judged.reasons[entry.Name] = entry.Reason
	}
	if err := checkNamesMatch(units, judged.verdicts, opts.File); err != nil {
		return round{}, err
	}
	return judged, nil
}

func verdictsOf(rounds []round) []map[string]bool {
	verdicts := make([]map[string]bool, len(rounds))
	for i, r := range rounds {
		verdicts[i] = r.verdicts
	}
	return verdicts
}

// collectReasons keeps the explanations for units that fell short of unanimity.
// A unit every run accepted has nothing to explain, and one entry per dissenting
// run is itself the tuning signal: four differently worded rejections say
// something that four identical ones do not.
func collectReasons(rounds []round, counts map[string]int, votes int) map[string][]string {
	reasons := map[string][]string{}
	for unit, count := range counts {
		if count == votes {
			continue
		}
		for _, r := range rounds {
			if r.verdicts[unit] {
				continue
			}
			if reason := strings.TrimSpace(r.reasons[unit]); reason != "" {
				reasons[unit] = append(reasons[unit], reason)
			}
		}
	}
	if len(reasons) == 0 {
		return nil
	}
	return reasons
}

func findDuplicate(names []string) (string, bool) {
	seen := make(map[string]bool, len(names))
	for _, name := range names {
		if seen[name] {
			return name, true
		}
		seen[name] = true
	}
	return "", false
}

func checkNamesMatch(names []string, verdicts map[string]bool, file string) error {
	listed := make(map[string]bool, len(names))
	for _, name := range names {
		listed[name] = true
	}

	var missing []string
	for _, name := range names {
		if _, hasVerdict := verdicts[name]; !hasVerdict {
			missing = append(missing, name)
		}
	}
	var unexpected []string
	for name := range verdicts {
		if !listed[name] {
			unexpected = append(unexpected, name)
		}
	}
	sort.Strings(unexpected)

	var problems []string
	if len(missing) > 0 {
		problems = append(problems, "missing "+strings.Join(missing, ", "))
	}
	if len(unexpected) > 0 {
		problems = append(problems, "unexpected "+strings.Join(unexpected, ", "))
	}
	if len(problems) == 0 {
		return nil
	}
	return fmt.Errorf("verdicts for %s do not cover exactly the test functions listed: %s", file, strings.Join(problems, "; "))
}

func formatFileBlock(f SourceFile) string {
	return fmt.Sprintf("=== FILE: %s ===\n%s\n=== END FILE: %s ===\n", f.Path, f.Content, f.Path)
}
