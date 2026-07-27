package lint

import (
	"context"
	"encoding/json"
	"fmt"
	"hash/fnv"
	"os"
	"slices"
	"sort"
	"strings"
	"unicode"

	"github.com/matthijn/aritu/internal/domain/rule"
	"github.com/matthijn/aritu/internal/lib/claudecli"
	"github.com/matthijn/aritu/internal/lib/vote"
	"github.com/matthijn/aritu/prompts"
)

// Exit is a process exit status. A split vote is a rule failure, never a
// could-not-run: routing it to ExitError would invite a commit hook to treat an
// unsure model as a tooling problem and skip past exactly the test this tool
// exists to catch.
type Exit int

// Unit is one judged thing: Name is the identifier a reader sees when it fails,
// Key is what the model answers under. They differ because a key has to be a tidy
// JSON property while a name has to stay exactly what CI prints.
type Unit struct {
	Name string
	Key  string
}

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

// Apply votes on one file against one rule. Everything the rule reads is read
// before anything is asked, so a target the rule cannot see costs no model call.
// The returned Report carries Rule, File and Votes even when the error is
// non-nil, so the caller can always emit output before exiting.
func Apply(ctx context.Context, ask claudecli.Ask, opts Options) (Report, error) {
	report := Report{Rule: opts.Rule.Name, File: opts.File, Votes: opts.Votes}
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

// Enumerate lists a file's leaves, always at test granularity. It is deliberately
// independent of the rule: the enumeration prompt is built from the granularity and
// the file alone, so every rule over one file would otherwise ask the same question
// and pay for the same answer. Coarser levels roll up from this with UnitsAt.
func Enumerate(ctx context.Context, ask claudecli.Ask, opts Options) ([]string, error) {
	test, err := readSourceFile(opts.File)
	if err != nil {
		return nil, err
	}
	enumerating := opts
	enumerating.Rule.Granularity = rule.GranularityTest
	return askNames(ctx, ask, enumerating, test)
}

// NeedsEnumeration reports whether the model has to list a rule's units at all.
// At file granularity the unit is the path, which costs no tokens to know and
// cannot be disagreed with, so no enumeration is asked for.
func NeedsEnumeration(granularity rule.Granularity) bool {
	return granularity != rule.GranularityFile
}

// UnitsAt narrows a file's leaves to the units one rule judges. A test that
// declares no cases appears in the leaf list as its bare name and one that declares
// cases appears once per case, so the distinct halves in front of the case are
// exactly the set of tests — the roll-up is a string split rather than a second
// question.
func UnitsAt(granularity rule.Granularity, file string, leaves []string) []Unit {
	switch granularity {
	case rule.GranularityFile:
		return UnitsFor([]string{file})
	case rule.GranularityFunction:
		return UnitsFor(distinctFunctions(leaves))
	case rule.GranularityTest:
		return UnitsFor(leaves)
	default:
		panic(fmt.Sprintf("unknown granularity: %d", int(granularity)))
	}
}

// Judge votes on units already enumerated. The returned Report carries Rule, File
// and Votes even when the error is non-nil, so the caller can always emit output
// before exiting.
func Judge(ctx context.Context, ask claudecli.Ask, opts Options, units []Unit) (Report, error) {
	report := Report{Rule: opts.Rule.Name, File: opts.File, Votes: opts.Votes}
	if opts.Votes < 1 {
		return report, fmt.Errorf("votes must be at least 1, got %d", opts.Votes)
	}
	files, err := readFiles(opts.Rule, opts.File)
	if err != nil {
		return report, err
	}
	return voteOn(ctx, ask, opts, files, units)
}

func leavesFor(ctx context.Context, ask claudecli.Ask, opts Options) ([]string, error) {
	if !NeedsEnumeration(opts.Rule.Granularity) {
		return nil, nil
	}
	return Enumerate(ctx, ask, opts)
}

func voteOn(ctx context.Context, ask claudecli.Ask, opts Options, files []SourceFile, units []Unit) (Report, error) {
	report := Report{Rule: opts.Rule.Name, File: opts.File, Votes: opts.Votes}

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

func distinctFunctions(leaves []string) []string {
	functions := make([]string, 0, len(leaves))
	seen := make(map[string]bool, len(leaves))
	for _, leaf := range leaves {
		function, _, _ := splitUnit(leaf)
		if seen[function] {
			continue
		}
		seen[function] = true
		functions = append(functions, function)
	}
	return functions
}

// ExitFor derives the exit status of a completed report.
func ExitFor(r Report) Exit {
	if vote.IsUnanimous(r.Verdicts, r.Votes) {
		return ExitPass
	}
	return ExitFail
}

// BuildNamesPrompt asks the model to enumerate a file's units. It is never called
// at file granularity, where the unit is the path and no model is needed to know it.
func BuildNamesPrompt(granularity rule.Granularity, source SourceFile) string {
	if !NeedsEnumeration(granularity) {
		panic(fmt.Sprintf("no names prompt for granularity: %s", granularity))
	}
	return prompts.Enumerate(formatFileBlock(source))
}

// BuildVerdictPrompt frames one rule's criterion around the files under judgement
// and the exact units to judge. The units are listed rather than left to be
// re-derived: two independent enumerations of twenty-five table rows will phrase one
// of them differently sooner or later, and every such disagreement would surface as
// a could-not-run.
func BuildVerdictPrompt(rulePrompt string, files []SourceFile, units []Unit) string {
	var listed strings.Builder
	for _, unit := range units {
		fmt.Fprintf(&listed, "- %s   ->   %s\n", unit.Name, unit.Key)
	}
	var sources strings.Builder
	for _, f := range files {
		sources.WriteString(formatFileBlock(f))
		sources.WriteString("\n")
	}
	return prompts.Verdict(rulePrompt, listed.String(), sources.String())
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

func askNames(ctx context.Context, ask claudecli.Ask, opts Options, test SourceFile) ([]string, error) {
	raw, err := ask(ctx, claudecli.Request{
		Prompt: BuildNamesPrompt(opts.Rule.Granularity, test),
		Model:  opts.Model,
		Effort: opts.Effort,
		Schema: json.RawMessage(NamesSchema),
	})
	if err != nil {
		return nil, fmt.Errorf("listing the tests in %s: %w", test.Path, err)
	}

	var reply namesReply
	if err := json.Unmarshal(raw, &reply); err != nil {
		return nil, fmt.Errorf("reading the test names for %s: %w", test.Path, err)
	}
	if duplicate, hasDuplicate := findDuplicate(reply.Names); hasDuplicate {
		return nil, fmt.Errorf("test unit %q listed more than once in %s", duplicate, test.Path)
	}
	return reply.Names, nil
}

func askVerdicts(ctx context.Context, ask claudecli.Ask, opts Options, files []SourceFile, units []Unit) (round, error) {
	raw, err := ask(ctx, claudecli.Request{
		Prompt: BuildVerdictPrompt(opts.Rule.Prompt, files, units),
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

// checkKeysMatch should never fire: the generated schema names every key and
// forbids any other, so the CLI rejects a non-conforming reply before it reaches
// here. It stays as an assertion against a contract this package does not own, and
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

func formatFileBlock(f SourceFile) string {
	return fmt.Sprintf("=== FILE: %s ===\n%s\n=== END FILE: %s ===\n", f.Path, f.Content, f.Path)
}

// UnitsFor derives the key each enumerated identifier is answered under.
func UnitsFor(names []string) []Unit {
	units := make([]Unit, 0, len(names))
	for _, name := range names {
		units = append(units, Unit{Name: name, Key: keyFor(name)})
	}
	return units
}

// VerdictSchemaFor names every key the reply may carry. An object cannot repeat a
// key, cannot omit a required one and cannot carry an extra one, so duplicated,
// dropped and invented units stop being errors this package has to detect and
// become schema violations the CLI retries on its own.
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
// string or a boolean it fails the CLI's strict validation, and the whole call
// then comes back as retries exhausted rather than as a rejected schema — a
// could-not-run that looks like an unreliable model.
type schemaNode struct {
	Type                 string                `json:"type"`
	Properties           map[string]schemaNode `json:"properties,omitempty"`
	Required             []string              `json:"required,omitempty"`
	AdditionalProperties *bool                 `json:"additionalProperties,omitempty"`
}

// closedObject is an object that may carry no key beyond the ones named, which is
// what turns a duplicated, dropped or invented unit into a schema violation the
// CLI retries rather than an error this package has to detect.
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

// keyFor derives the property a unit answers under: a digest of the whole name,
// then a normalised form of the name a reader can recognise.
//
// Uniqueness rides entirely on the digest, which is what lets the readable half be
// cut to fit the API's ceiling. Cutting a readable key on its own is the wrong
// answer twice over: the prefix that survives is neither unique — two files under
// one long directory reduce to the same string — nor legible, and dropping a unit's
// own property would hand it a neighbour's verdict with every count still looking
// healthy.
func keyFor(name string) string {
	digest := fmt.Sprintf("%08x", fnv1aOf(name))
	readable := fitParts(partsOf(name), maxKeyLength-len(digest)-1)
	if readable == "" {
		return digest
	}
	return digest + "." + readable
}

// partsOf breaks a unit name into the parts a key joins with ".": the segments of a
// path, the enclosing scopes of a test, and the case in the trailing parenthesis.
// Each part is normalised on its own, so a separator in the source can never
// survive as a run of underscores in the key.
func partsOf(name string) []string {
	function, caseName, hasCase := splitUnit(name)
	raw := strings.FieldsFunc(function, isPartBoundary)
	if hasCase {
		raw = append(raw, caseName)
	}
	parts := make([]string, 0, len(raw))
	for _, part := range raw {
		if normalised := snakeCase(part); normalised != "" {
			parts = append(parts, normalised)
		}
	}
	return parts
}

func isPartBoundary(r rune) bool {
	return r == '/' || r == '\\' || r == '>'
}

// fitParts keeps as many whole trailing parts as the budget allows. The tail is the
// half a reader recognises — the file name, the case that failed — so whole parts
// are dropped from the front before a single part is cut at all, and a part is only
// opened mid-word when one part alone is over the budget.
func fitParts(parts []string, budget int) string {
	if len(parts) == 0 || budget <= 0 {
		return ""
	}
	kept := 0
	length := 0
	for _, part := range slices.Backward(parts) {
		grown := length + len(part)
		if kept > 0 {
			grown++
		}
		if grown > budget {
			break
		}
		length = grown
		kept++
	}
	if kept == 0 {
		return trimSeparators(lastChars(parts[len(parts)-1], budget))
	}
	return strings.Join(parts[len(parts)-kept:], ".")
}

func fnv1aOf(text string) uint32 {
	digest := fnv.New32a()
	digest.Write([]byte(text))
	return digest.Sum32()
}

func lastChars(text string, count int) string {
	if len(text) <= count {
		return text
	}
	return text[len(text)-count:]
}

func trimSeparators(key string) string {
	return strings.Trim(key, keySeparators)
}

// maxKeyLength and the character set below are the API's, not ours: a schema
// property key is rejected outright unless it matches ^[a-zA-Z0-9_.-]{1,64}$.
// Colons, slashes and spaces are all out, which is why a unit's key cannot simply
// be the identifier a reader sees.
const maxKeyLength = 64

// keySeparators may sit inside a key but never at its end. The CLI derives one
// tool parameter per top-level schema property, and a name at the ceiling that
// ends on one fails that derivation: the parameter becomes an unsubstituted
// placeholder, and no reply can satisfy it however correct the answer.
//
// Each separator carries one level of the name: "." between the parts, "_" between
// the words of one part. A colon would read better in front of the digest, but the
// character set above does not permit one.
const keySeparators = "_-."

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

// snakeCase normalises one part. Camel and acronym boundaries become word breaks,
// anything outside the key's character set collapses to a single underscore however
// much of it there was, and no underscore survives at either end. Normalising a
// part on its own is what keeps "a > b" from reaching the key as three separators.
func snakeCase(text string) string {
	var b strings.Builder
	runes := []rune(text)
	pendingSeparator := false
	for at, r := range runes {
		if !isWordRune(r) {
			pendingSeparator = true
			continue
		}
		if b.Len() > 0 && (pendingSeparator || startsWord(runes, at)) {
			b.WriteByte('_')
		}
		pendingSeparator = false
		b.WriteRune(unicode.ToLower(r))
	}
	return b.String()
}

// startsWord reports whether the rune at an index opens a new word inside a run of
// letters, so that ParseHTTPHeader breaks into parse, http and header rather than
// reaching the key as one unreadable word.
//
// A capital followed by a lower-case letter closes whatever ran into it, which is
// what separates the acronym in ParseHTTPHeader from the word after it. The same
// rule splits IPv6 into i_pv6, and that is the better trade: telling those apart
// needs a dictionary, and the alternative merges the far more common DoesAThing
// into does_athing.
func startsWord(runes []rune, at int) bool {
	if at == 0 || !unicode.IsUpper(runes[at]) {
		return false
	}
	previous := runes[at-1]
	if unicode.IsLower(previous) || unicode.IsDigit(previous) {
		return true
	}
	return unicode.IsUpper(previous) && at+1 < len(runes) && unicode.IsLower(runes[at+1])
}

// isWordRune is deliberately ASCII: a rune outside this set becomes a separator
// rather than being lowercased into the key, which is what guarantees every key
// matches the character set the API accepts.
func isWordRune(r rune) bool {
	return r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9'
}
