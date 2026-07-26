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

// UnitsAt narrows a file's leaves to the units one rule judges. A function that
// declares no cases appears in the leaf list as its bare name and one that declares
// cases appears once per case, so the distinct function halves are exactly the set
// of test functions — the roll-up is a string split rather than a second question.
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
func BuildVerdictPrompt(base, rulePrompt string, files []SourceFile, units []Unit) string {
	var b strings.Builder
	if trimmed := strings.TrimSpace(base); trimmed != "" {
		b.WriteString(trimmed)
		b.WriteString("\n\n---\n\n")
	}
	b.WriteString(strings.TrimSpace(rulePrompt))
	b.WriteString("\n\n---\n\n")
	b.WriteString("Judge exactly these units against the rule above. Each line gives the unit, then the key to answer under:\n")
	for _, unit := range units {
		b.WriteString(fmt.Sprintf("- %s   ->   %s\n", unit.Name, unit.Key))
	}
	b.WriteString("\nJudge the unit as written on the left. The key on the right is only where the answer goes.\n\n")
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

func askVerdicts(ctx context.Context, ask claudecli.Ask, opts Options, files []SourceFile, units []Unit) (round, error) {
	raw, err := ask(ctx, claudecli.Request{
		Prompt: BuildVerdictPrompt(opts.Base, opts.Rule.Prompt, files, units),
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

// UnitsFor derives the key each enumerated identifier is answered under. The test
// function name is kept verbatim and only the case name is normalised, because the
// case is the half carrying arbitrary prose.
//
// Two cases in one function can normalise alike — "empty input" and "empty  input"
// both reach empty_input, and truncation to the API's 64-character ceiling creates
// more. Left alone the second would overwrite the first while the
// schema is built, so a unit would vanish from the run with every count still
// looking healthy. Suffixing mirrors what Go does for duplicate subtest names.
func UnitsFor(names []string) []Unit {
	units := make([]Unit, 0, len(names))
	taken := make(map[string]int, len(names))
	for _, name := range names {
		key := keyFor(name)
		if seen := taken[key]; seen > 0 {
			key = fmt.Sprintf("%s-%02d", truncateKey(key, maxKeyLength-3), seen)
		}
		taken[keyFor(name)]++
		units = append(units, Unit{Name: name, Key: key})
	}
	return units
}

// VerdictSchemaFor names every key the reply may carry. An object cannot repeat a
// key, cannot omit a required one and cannot carry an extra one, so duplicated,
// dropped and invented units stop being errors this package has to detect and
// become schema violations the CLI retries on its own.
func VerdictSchemaFor(units []Unit) json.RawMessage {
	schema := objectSchema{
		Type:                 "object",
		Properties:           make(map[string]objectSchema, len(units)),
		Required:             make([]string, 0, len(units)),
		AdditionalProperties: false,
	}
	for _, unit := range units {
		schema.Properties[unit.Key] = answerSchema()
		schema.Required = append(schema.Required, unit.Key)
	}
	encoded, err := json.Marshal(schema)
	if err != nil {
		panic(fmt.Sprintf("the verdict schema failed to marshal, which its types make impossible: %v", err))
	}
	return encoded
}

type objectSchema struct {
	Type                 string                  `json:"type"`
	Properties           map[string]objectSchema `json:"properties,omitempty"`
	Required             []string                `json:"required,omitempty"`
	AdditionalProperties bool                    `json:"additionalProperties"`
}

func answerSchema() objectSchema {
	return objectSchema{
		Type: "object",
		Properties: map[string]objectSchema{
			"satisfies": {Type: "boolean"},
			"reason":    {Type: "string"},
		},
		Required:             []string{"satisfies", "reason"},
		AdditionalProperties: false,
	}
}

func keyFor(name string) string {
	function, caseName, hasCase := splitUnit(name)
	key := sanitiseKey(function)
	if hasCase {
		normalised := snakeCase(caseName)
		if normalised == "" {
			normalised = "case"
		}
		key += "." + normalised
	}
	if key == "" {
		key = "unit"
	}
	return truncateKey(key, maxKeyLength)
}

// maxKeyLength and the character set below are the API's, not ours: a schema
// property key is rejected outright unless it matches ^[a-zA-Z0-9_.-]{1,64}$.
// Colons, slashes and spaces are all out, which is why a unit's key cannot simply
// be the identifier a reader sees.
const maxKeyLength = 64

func sanitiseKey(text string) string {
	var b strings.Builder
	for _, r := range text {
		if isKeyRune(r) {
			b.WriteRune(r)
			continue
		}
		b.WriteByte('_')
	}
	return b.String()
}

func isKeyRune(r rune) bool {
	switch {
	case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9':
		return true
	case r == '_', r == '.', r == '-':
		return true
	default:
		return false
	}
}

func truncateKey(key string, limit int) string {
	if len(key) <= limit {
		return key
	}
	return key[:limit]
}

func splitUnit(name string) (function, caseName string, hasCase bool) {
	open := strings.Index(name, " (")
	if open < 0 || !strings.HasSuffix(name, ")") {
		return name, "", false
	}
	return name[:open], name[open+2 : len(name)-1], true
}

func snakeCase(text string) string {
	var b strings.Builder
	pendingSeparator := false
	for _, r := range strings.ToLower(text) {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			if pendingSeparator && b.Len() > 0 {
				b.WriteByte('_')
			}
			pendingSeparator = false
			b.WriteRune(r)
		default:
			pendingSeparator = true
		}
	}
	return b.String()
}
