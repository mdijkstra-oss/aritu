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

type Exit int

type Unit = prompts.Unit

type SourceFile = prompts.File

type Report struct {
	Rule     string              `json:"rule"`
	Priority string              `json:"priority,omitempty"`
	File     string              `json:"file"`
	Votes    int                 `json:"votes"`
	Verdicts map[string]int      `json:"verdicts"`
	Reasons  map[string][]string `json:"reasons,omitempty"`
	Error    string              `json:"error,omitempty"`
}

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

func ReportFor(opts Options) Report {
	return Report{
		Rule:     opts.Rule.Name,
		Priority: opts.Rule.Priority.String(),
		File:     opts.File,
		Votes:    opts.Votes,
	}
}

func Apply(ctx context.Context, ask service.Ask, opts Options) (Report, error) {
	report := ReportFor(opts)
	files, err := filesToJudge(opts)
	if err != nil {
		return report, err
	}
	leaves, err := leavesFor(ctx, ask, opts)
	if err != nil {
		return report, err
	}
	return voteOn(ctx, ask, ballot{opts: opts, files: files, units: UnitsAt(opts.Rule.Granularity, opts.File, leaves)})
}

func Enumerate(ctx context.Context, ask service.Ask, opts Options) ([]string, error) {
	file, err := readSourceFile(opts.File)
	if err != nil {
		return nil, err
	}
	return askNames(ctx, ask, opts, file)
}

func NeedsEnumeration(granularity rule.Granularity) bool {
	return granularity != rule.GranularityFile
}

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

func Judge(ctx context.Context, ask service.Ask, opts Options, units []Unit) (Report, error) {
	report := ReportFor(opts)
	files, err := filesToJudge(opts)
	if err != nil {
		return report, err
	}
	return voteOn(ctx, ask, ballot{opts: opts, files: files, units: units})
}

func filesToJudge(opts Options) ([]SourceFile, error) {
	if opts.Votes < 1 {
		return nil, fmt.Errorf("votes must be at least 1, got %d", opts.Votes)
	}
	return readFiles(opts.Rule, opts.File)
}

func leavesFor(ctx context.Context, ask service.Ask, opts Options) ([]string, error) {
	if !NeedsEnumeration(opts.Rule.Granularity) {
		return nil, nil
	}
	return Enumerate(ctx, ask, opts)
}

type ballot struct {
	opts  Options
	files []SourceFile
	units []Unit
}

func (b ballot) withUnits(units []Unit) ballot {
	b.units = units
	return b
}

func voteOn(ctx context.Context, ask service.Ask, cast ballot) (Report, error) {
	report := ReportFor(cast.opts)

	judge := func(ctx context.Context, _ int) (round, error) {
		return askVerdicts(ctx, ask, cast)
	}
	rounds, err := vote.Collect(ctx, cast.opts.Votes, judge)
	if err != nil {
		return report, err
	}

	report.Verdicts = vote.Tally(verdictsOf(rounds))
	report.Reasons = collectReasons(rounds, report.Verdicts, cast.opts.Votes)
	return report, nil
}

func ExitFor(r Report) Exit {
	for _, count := range r.Verdicts {
		if OutcomeFor(count, r.Votes) != OutcomePass {
			return ExitFail
		}
	}
	return ExitPass
}

func BuildNamesPrompt(granularity rule.Granularity, source SourceFile) string {
	if !NeedsEnumeration(granularity) {
		panic(fmt.Sprintf("no names prompt for granularity: %s", granularity))
	}
	return prompts.Splitter(granularity.String(), source)
}

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

func askVerdicts(ctx context.Context, ask service.Ask, cast ballot) (round, error) {
	judged := round{
		verdicts: make(map[string]bool, len(cast.units)),
		reasons:  make(map[string]string, len(cast.units)),
	}
	for _, batch := range batchesOf(cast.units, maxUnitsPerCall) {
		answers, err := askBatch(ctx, ask, cast.withUnits(batch))
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

func askBatch(ctx context.Context, ask service.Ask, cast ballot) (map[string]verdictAnswer, error) {
	opts := cast.opts
	raw, err := ask(ctx, service.Request{
		Prompt: BuildVerdictPrompt(opts.Rule, cast.files, cast.units),
		Model:  opts.Model,
		Effort: opts.Effort,
		Schema: VerdictSchemaFor(cast.units),
	})
	if err != nil {
		return nil, fmt.Errorf("judging %s against rule %s: %w", opts.File, opts.Rule.Name, err)
	}

	answers := map[string]verdictAnswer{}
	if err := json.Unmarshal(raw, &answers); err != nil {
		return nil, fmt.Errorf("reading verdicts for %s: %w", opts.File, err)
	}
	if err := checkKeysMatch(cast.units, answers, opts.File); err != nil {
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
