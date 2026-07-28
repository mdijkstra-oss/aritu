package lint

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/matthijn/aritu/internal/domain/rule"
	"github.com/matthijn/aritu/internal/lib/service"
	"github.com/matthijn/aritu/internal/lib/vote"
	"github.com/matthijn/aritu/prompts"
)

// Exit is a process exit status. A split vote is a rule failure, never a
// could-not-run: routing it to ExitError would invite a commit hook to treat an
// unsure model as a tooling problem and skip past exactly the test this tool
// exists to catch.
type Exit int

// Unit is one judged thing, as the prompts package defines it: the name a reader
// sees and the key the model answers under. The alias keeps it one type, so a
// unit built here is a unit rendered there without a copy in between.
type Unit = prompts.Unit

// SourceFile is a file's path and contents as handed to the model.
type SourceFile = prompts.File

// Report is the tool's output. Verdicts maps each judged unit to how many of
// Votes runs judged it to satisfy the rule; a strict majority passes and a tie
// fails. Reasons carries one explanation per dissenting run, for units that
// fell short.
type Report struct {
	Rule     string              `json:"rule"`
	Priority string              `json:"priority,omitempty"`
	File     string              `json:"file"`
	Votes    int                 `json:"votes"`
	Verdicts map[string]int      `json:"verdicts"`
	Reasons  map[string][]string `json:"reasons,omitempty"`
	Error    string              `json:"error,omitempty"`
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

// ReportFor fills the header before anything is judged, because a target that
// fails on its first model call still has to print one.
func ReportFor(opts Options) Report {
	return Report{
		Rule:     opts.Rule.Name,
		Priority: opts.Rule.Priority.String(),
		File:     opts.File,
		Votes:    opts.Votes,
	}
}

// Apply votes on one file against one rule. Everything the rule reads is read
// before anything is asked, so a target the rule cannot see costs no model call.
// The returned Report carries its header even when the error is non-nil, so the
// caller can always emit output before exiting.
func Apply(ctx context.Context, ask service.Ask, opts Options) (Report, error) {
	report := ReportFor(opts)
	if opts.Votes < 1 {
		return report, fmt.Errorf("votes must be at least 1, got %d", opts.Votes)
	}
	files, err := readFiles(opts.Rule, opts.File)
	if err != nil {
		return report, err
	}
	leaves, err := leavesFor(ctx, ask, opts)
	if err != nil {
		return report, err
	}
	return voteOn(ctx, ask, opts, files, UnitsAt(opts.Rule.Granularity, opts.File, leaves))
}

// Enumerate lists a file's units at the rule's own granularity. It is
// deliberately independent of the rule's text: the splitter prompt is built from
// the granularity and the file alone, so every rule judging one file at one
// level asks the same question and can share the same answer.
func Enumerate(ctx context.Context, ask service.Ask, opts Options) ([]string, error) {
	file, err := readSourceFile(opts.File)
	if err != nil {
		return nil, err
	}
	return askNames(ctx, ask, opts, file)
}

// NeedsEnumeration reports whether the model has to list a rule's units at all.
// At file granularity the unit is the path, which costs no tokens to know and
// cannot be disagreed with, so no enumeration is asked for.
func NeedsEnumeration(granularity rule.Granularity) bool {
	return granularity != rule.GranularityFile
}

// UnitsAt derives the units one rule judges. At file granularity the unit is the
// path; at every other level it is whatever the splitter listed, because each
// granularity asked for its own kind of unit and got exactly that.
func UnitsAt(granularity rule.Granularity, file string, leaves []string) []Unit {
	switch granularity {
	case rule.GranularityFile:
		return UnitsFor([]string{file})
	case rule.GranularityFunction, rule.GranularityTestCase, rule.GranularityComment, rule.GranularityDeclaration:
		return UnitsFor(leaves)
	default:
		panic(fmt.Sprintf("unknown granularity: %d", int(granularity)))
	}
}

// Judge votes on units already enumerated. The returned Report carries its
// header even when the error is non-nil, so the caller can always emit output
// before exiting.
func Judge(ctx context.Context, ask service.Ask, opts Options, units []Unit) (Report, error) {
	report := ReportFor(opts)
	if opts.Votes < 1 {
		return report, fmt.Errorf("votes must be at least 1, got %d", opts.Votes)
	}
	files, err := readFiles(opts.Rule, opts.File)
	if err != nil {
		return report, err
	}
	return voteOn(ctx, ask, opts, files, units)
}

func leavesFor(ctx context.Context, ask service.Ask, opts Options) ([]string, error) {
	if !NeedsEnumeration(opts.Rule.Granularity) {
		return nil, nil
	}
	return Enumerate(ctx, ask, opts)
}

func voteOn(ctx context.Context, ask service.Ask, opts Options, files []SourceFile, units []Unit) (Report, error) {
	report := ReportFor(opts)

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
	for _, count := range r.Verdicts {
		if OutcomeFor(count, r.Votes) != OutcomePass {
			return ExitFail
		}
	}
	return ExitPass
}

// BuildNamesPrompt asks the model to list a file's units of one rule's kind. It
// is never called at file granularity, where the unit is the path and no model is
// needed to know it.
func BuildNamesPrompt(granularity rule.Granularity, source SourceFile) string {
	if !NeedsEnumeration(granularity) {
		panic(fmt.Sprintf("no names prompt for granularity: %s", granularity))
	}
	return prompts.Splitter(granularity.String(), source)
}

// BuildVerdictPrompt frames one rule's criterion around the files under judgement
// and the exact units to judge. The units are listed rather than left to be
// re-derived: two independent enumerations of twenty-five table rows will phrase one
// of them differently sooner or later, and every such disagreement would surface as
// a could-not-run.
//
// The rule reaches the model as rule.Section renders it, which is the same block
// the rulebook hands a writer, heading and all. Judging against a differently
// worded copy of the standard somebody was given is how a rule ends up meaning two
// things, so there is only ever the one rendering.
func BuildVerdictPrompt(judged rule.Rule, files []SourceFile, units []Unit) string {
	return prompts.Linter(judged.Granularity.String(), rule.Section(judged), units, files)
}

type namesReply struct {
	Names []string `json:"names"`
}

type verdictAnswer struct {
	Satisfies bool   `json:"satisfies"`
	Reason    string `json:"reason"`
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

	sourcePath, err := rule.FindSource(testPath)
	if err != nil {
		return nil, fmt.Errorf("rule %s needs the file under test: %w", r.Name, err)
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

func askNames(ctx context.Context, ask service.Ask, opts Options, file SourceFile) ([]string, error) {
	raw, err := ask(ctx, service.Request{
		Prompt: BuildNamesPrompt(opts.Rule.Granularity, file),
		Model:  opts.Model,
		Effort: opts.Effort,
		Schema: json.RawMessage(NamesSchema),
	})
	if err != nil {
		return nil, fmt.Errorf("listing the units in %s: %w", file.Path, err)
	}

	var reply namesReply
	if err := json.Unmarshal(raw, &reply); err != nil {
		return nil, fmt.Errorf("reading the unit names for %s: %w", file.Path, err)
	}
	return uniqueNames(reply.Names), nil
}

// maxUnitsPerCall bounds how many units one verdict call carries. The schema
// names a property per unit, and the endpoint rejects the request outright once
// that object grows too large: a file of seventy short-keyed units was accepted
// where eighty were not. Keys run to maxKeyLength, so the ceiling is lower for a
// file whose names are long, and the bound below is set well under the measured
// edge rather than at it.
//
// Batching costs a call per batch and buys nothing else: keys are assigned
// across the whole listing before it is cut up, so each unit answers under the
// same property it would have had in one call.
const maxUnitsPerCall = 50

func askVerdicts(ctx context.Context, ask service.Ask, opts Options, files []SourceFile, units []Unit) (round, error) {
	judged := round{
		verdicts: make(map[string]bool, len(units)),
		reasons:  make(map[string]string, len(units)),
	}
	for _, batch := range batchesOf(units, maxUnitsPerCall) {
		answers, err := askBatch(ctx, ask, opts, files, batch)
		if err != nil {
			return round{}, err
		}
		for _, unit := range batch {
			answer := answers[unit.Key]
			judged.verdicts[unit.Name] = answer.Satisfies
			judged.reasons[unit.Name] = answer.Reason
		}
	}
	return judged, nil
}

func askBatch(ctx context.Context, ask service.Ask, opts Options, files []SourceFile, units []Unit) (map[string]verdictAnswer, error) {
	raw, err := ask(ctx, service.Request{
		Prompt: BuildVerdictPrompt(opts.Rule, files, units),
		Model:  opts.Model,
		Effort: opts.Effort,
		Schema: VerdictSchemaFor(units),
	})
	if err != nil {
		return nil, fmt.Errorf("judging %s against rule %s: %w", opts.File, opts.Rule.Name, err)
	}

	answers := map[string]verdictAnswer{}
	if err := json.Unmarshal(raw, &answers); err != nil {
		return nil, fmt.Errorf("reading verdicts for %s: %w", opts.File, err)
	}
	if err := checkKeysMatch(units, answers, opts.File); err != nil {
		return nil, err
	}
	return answers, nil
}

func batchesOf(units []Unit, size int) [][]Unit {
	batches := make([][]Unit, 0, (len(units)+size-1)/size)
	for start := 0; start < len(units); start += size {
		batches = append(batches, units[start:min(start+size, len(units))])
	}
	return batches
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

// uniqueNames collapses repeats while keeping first-listed order. A repeated
// name is not necessarily the model misbehaving — a file can hold two methods
// called Help — so the repeats fold into one judged unit rather than refusing
// the run.
func uniqueNames(names []string) []string {
	seen := make(map[string]bool, len(names))
	unique := make([]string, 0, len(names))
	for _, name := range names {
		if seen[name] {
			continue
		}
		seen[name] = true
		unique = append(unique, name)
	}
	return unique
}

// checkKeysMatch should never fire: the generated schema names every key and
// forbids any other, so the endpoint's strict json_schema rejects a non-conforming
// reply before it reaches here. It stays as an assertion against a contract this package does not own, and
// keeps exit 2, because a schema that failed to hold is a could-not-run.
func checkKeysMatch(units []Unit, answers map[string]verdictAnswer, file string) error {
	expected := make(map[string]bool, len(units))
	var missing []string
	for _, unit := range units {
		expected[unit.Key] = true
		if _, answered := answers[unit.Key]; !answered {
			missing = append(missing, unit.Key)
		}
	}
	var unexpected []string
	for key := range answers {
		if !expected[key] {
			unexpected = append(unexpected, key)
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
	return fmt.Errorf("verdicts for %s do not cover exactly the units listed: %s", file, strings.Join(problems, "; "))
}
