package lint

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"
	"unicode"

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

// NamesSchema constrains the test-name call. Its keys are fixed rather than
// generated because the answer's shape never varies: one array, however many tests
// the file holds.
const NamesSchema = `{"type":"object","properties":{"names":{"type":"array","items":{"type":"string"}}},"required":["names"],"additionalProperties":false}`

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
	case rule.GranularityFunction, rule.GranularityTestCase:
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

func askVerdicts(ctx context.Context, ask service.Ask, opts Options, files []SourceFile, units []Unit) (round, error) {
	raw, err := ask(ctx, service.Request{
		Prompt: BuildVerdictPrompt(opts.Rule, files, units),
		Model:  opts.Model,
		Effort: opts.Effort,
		Schema: VerdictSchemaFor(units),
	})
	if err != nil {
		return round{}, fmt.Errorf("judging %s against rule %s: %w", opts.File, opts.Rule.Name, err)
	}

	answers := map[string]verdictAnswer{}
	if err := json.Unmarshal(raw, &answers); err != nil {
		return round{}, fmt.Errorf("reading verdicts for %s: %w", opts.File, err)
	}
	if err := checkKeysMatch(units, answers, opts.File); err != nil {
		return round{}, err
	}

	judged := round{
		verdicts: make(map[string]bool, len(units)),
		reasons:  make(map[string]string, len(units)),
	}
	for _, unit := range units {
		answer := answers[unit.Key]
		judged.verdicts[unit.Name] = answer.Satisfies
		judged.reasons[unit.Name] = answer.Reason
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

// UnitsFor derives the key each enumerated identifier is answered under.
func UnitsFor(names []string) []Unit {
	units := make([]Unit, 0, len(names))
	for at, name := range names {
		units = append(units, Unit{Name: name, Key: keyFor(at, name)})
	}
	return units
}

// VerdictSchemaFor names every key the reply may carry. An object cannot repeat a
// key, cannot omit a required one and cannot carry an extra one, so duplicated,
// dropped and invented units stop being errors this package has to detect and
// become schema violations the endpoint refuses to produce.
func VerdictSchemaFor(units []Unit) json.RawMessage {
	answers := make(map[string]schemaNode, len(units))
	keys := make([]string, 0, len(units))
	for _, unit := range units {
		answers[unit.Key] = answerSchema()
		keys = append(keys, unit.Key)
	}
	encoded, err := json.Marshal(closedObject(answers, keys))
	if err != nil {
		panic(fmt.Sprintf("the verdict schema failed to marshal, which its types make impossible: %v", err))
	}
	return encoded
}

// schemaNode is one node of a generated JSON Schema. AdditionalProperties is a
// pointer because the keyword only means anything on an object: emitted beside a
// string or a boolean the endpoint rejects the request outright, and a rejected
// request is not retried — every call carrying the schema fails the same way, for
// a reason that reads as the endpoint's rather than as ours.
type schemaNode struct {
	Type                 string                `json:"type"`
	Properties           map[string]schemaNode `json:"properties,omitempty"`
	Required             []string              `json:"required,omitempty"`
	AdditionalProperties *bool                 `json:"additionalProperties,omitempty"`
}

// closedObject is an object that may carry no key beyond the ones named, which is
// what turns a duplicated, dropped or invented unit into a reply the endpoint's
// strict enforcement will not produce, rather than an error this package detects.
func closedObject(properties map[string]schemaNode, required []string) schemaNode {
	isClosed := false
	return schemaNode{
		Type:                 "object",
		Properties:           properties,
		Required:             required,
		AdditionalProperties: &isClosed,
	}
}

func answerSchema() schemaNode {
	return closedObject(map[string]schemaNode{
		"satisfies": {Type: "boolean"},
		"reason":    {Type: "string"},
	}, []string{"satisfies", "reason"})
}

// keyFor derives the property a unit answers under: its position in the listed
// units, then a normalised form of the name a reader can recognise.
//
// Uniqueness rides entirely on the position, which is what lets the readable half
// be cut to fit the API's ceiling. Cutting a readable key on its own is the wrong
// answer twice over: the prefix that survives is neither unique — two files under
// one long directory reduce to the same string — nor legible, and dropping a unit's
// own property would hand it a neighbour's verdict with every count still looking
// healthy. The tail is kept over the head because it is the half a reader
// recognises: the file name, the case that failed.
func keyFor(at int, name string) string {
	prefix := fmt.Sprintf("u%02d", at+1)
	readable := strings.Trim(lastChars(snakeCase(name), maxKeyLength-len(prefix)-1), "_")
	if readable == "" {
		return prefix
	}
	return prefix + "_" + readable
}

// maxKeyLength and the character set below are the API's, not ours: a schema
// property key is rejected outright unless it matches ^[a-zA-Z0-9_.-]{1,64}$.
// Colons, slashes and spaces are all out, which is why a unit's key cannot simply
// be the identifier a reader sees.
const maxKeyLength = 64

func lastChars(text string, count int) string {
	if len(text) <= count {
		return text
	}
	return text[len(text)-count:]
}

// splitUnit separates the test from the case a leaf name carries. It takes the
// last " (" rather than the first because the half in front of it is a namespace
// path of arbitrary prose — a grouping block may be named "Parser (v2)" — while the
// case is always the trailing parenthesis the enumeration appended.
func splitUnit(name string) (function, caseName string, hasCase bool) {
	open := strings.LastIndex(name, " (")
	if open < 0 || !strings.HasSuffix(name, ")") {
		return name, "", false
	}
	return name[:open], name[open+2 : len(name)-1], true
}

// snakeCase normalises a name for the key: anything outside the key's character
// set collapses to a single underscore however much of it there was, and a word
// break opens where a capital follows a lower-case letter or digit. Acronym-grade
// word splitting is deliberately absent — the position prefix already guarantees
// uniqueness, so the readable half only has to be recognisable.
func snakeCase(text string) string {
	var b strings.Builder
	pendingSeparator := false
	var previous rune
	for _, r := range text {
		if !isWordRune(r) {
			pendingSeparator = true
			continue
		}
		opensWord := unicode.IsUpper(r) && (unicode.IsLower(previous) || unicode.IsDigit(previous))
		if b.Len() > 0 && (pendingSeparator || opensWord) {
			b.WriteByte('_')
		}
		pendingSeparator = false
		previous = r
		b.WriteRune(unicode.ToLower(r))
	}
	return b.String()
}

// isWordRune is deliberately ASCII: a rune outside this set becomes a separator
// rather than being lowercased into the key, which is what guarantees every key
// matches the character set the API accepts.
func isWordRune(r rune) bool {
	return r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9'
}
