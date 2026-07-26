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

// Report is the tool's output. Verdicts maps each test function to how many of
// Votes runs judged it to satisfy the rule; only a count equal to Votes passes.
type Report struct {
	Rule     string         `json:"rule"`
	File     string         `json:"file"`
	Votes    int            `json:"votes"`
	Verdicts map[string]int `json:"verdicts"`
	Error    string         `json:"error,omitempty"`
}

// Options configures one Apply run.
type Options struct {
	Rule   rule.Rule
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

// VerdictSchema constrains the per-function verdict call.
const VerdictSchema = `{"type":"object","properties":{"results":{"type":"array","items":{"type":"object","properties":{"name":{"type":"string"},"satisfies":{"type":"boolean"}},"required":["name","satisfies"],"additionalProperties":false}}},"required":["results"],"additionalProperties":false}`

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

	names, err := askNames(ctx, ask, opts, files[0])
	if err != nil {
		return report, err
	}

	round := func(ctx context.Context, _ int) (map[string]bool, error) {
		return askVerdicts(ctx, ask, opts, files, names)
	}
	rounds, err := vote.Collect(ctx, opts.Votes, round)
	if err != nil {
		return report, err
	}

	report.Verdicts = vote.Tally(rounds)
	return report, nil
}

// ExitFor derives the exit status of a completed report.
func ExitFor(r Report) Exit {
	if vote.IsUnanimous(r.Verdicts, r.Votes) {
		return ExitPass
	}
	return ExitFail
}

// BuildNamesPrompt asks the model to enumerate the test functions in a file.
func BuildNamesPrompt(test SourceFile) string {
	var b strings.Builder
	b.WriteString("List every test function declared in the Go test file below.\n\n")
	b.WriteString("A test function is a top-level func whose name begins with Test and which takes a single *testing.T parameter.\n\n")
	b.WriteString("Do not list any of the following:\n")
	b.WriteString("- helper functions, including helpers that take *testing.T\n")
	b.WriteString("- struct types or literals holding table cases, and the names of those cases\n")
	b.WriteString("- subtest closures passed to t.Run, however they are named\n")
	b.WriteString("- benchmarks, fuzz targets and examples\n\n")
	b.WriteString("Report each name exactly as declared, in declaration order.\n")
	b.WriteString(`Answer with a JSON object holding a "names" array of strings.` + "\n\n")
	b.WriteString(formatFileBlock(test))
	return b.String()
}

// BuildVerdictPrompt frames a rule's prompt around the files under judgement.
func BuildVerdictPrompt(rulePrompt string, files []SourceFile) string {
	var b strings.Builder
	b.WriteString(strings.TrimSpace(rulePrompt))
	b.WriteString("\n\n")
	b.WriteString("Judge every test function in the files below against the rule above.\n\n")
	b.WriteString("- Every test function gets exactly one entry: its name, and whether it satisfies the rule. No entry for anything else, and never two entries for one name.\n")
	b.WriteString("- Test shapes vary. A table-driven test, a test built from subtests under t.Run, and a plain functional test are each one test function, whatever their internal structure.\n")
	b.WriteString("- Judge the behavior the test pins down, not the syntax it is written in. No shape is by itself a pass or a fail.\n")
	b.WriteString(`Answer with a JSON object holding a "results" array, each entry a "name" and a "satisfies" boolean.` + "\n\n")
	for _, f := range files {
		b.WriteString(formatFileBlock(f))
		b.WriteString("\n")
	}
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
	Satisfies bool   `json:"satisfies"`
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
		Prompt: BuildNamesPrompt(test),
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
		return nil, fmt.Errorf("test function %q listed more than once in %s", duplicate, test.Path)
	}
	return reply.Names, nil
}

func askVerdicts(ctx context.Context, ask claudecli.Ask, opts Options, files []SourceFile, names []string) (map[string]bool, error) {
	raw, err := ask(ctx, claudecli.Request{
		Prompt: BuildVerdictPrompt(opts.Rule.Prompt, files),
		Model:  opts.Model,
		Effort: opts.Effort,
		Schema: json.RawMessage(VerdictSchema),
	})
	if err != nil {
		return nil, fmt.Errorf("judging %s against rule %s: %w", opts.File, opts.Rule.Name, err)
	}

	var reply verdictReply
	if err := json.Unmarshal(raw, &reply); err != nil {
		return nil, fmt.Errorf("reading verdicts for %s: %w", opts.File, err)
	}

	verdicts := make(map[string]bool, len(reply.Results))
	for _, entry := range reply.Results {
		if _, isRepeat := verdicts[entry.Name]; isRepeat {
			return nil, fmt.Errorf("verdict for test function %q given twice in %s", entry.Name, opts.File)
		}
		verdicts[entry.Name] = entry.Satisfies
	}
	if err := checkNamesMatch(names, verdicts, opts.File); err != nil {
		return nil, err
	}
	return verdicts, nil
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
