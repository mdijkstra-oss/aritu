package rule

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/matthijn/aritu/prompts"

	"github.com/matthijn/aritu/internal/lib/testpath"
)

// Rule is one linting rule loaded from a directory under the rules dir.
type Rule struct {
	Name          string
	Dir           string
	Prompt        string
	Targets       []string
	Include       []string
	IncludeSource bool
	Granularity   Granularity
}

// Expectation is the pass/fail outcome a fixture directory name asserts.
type Expectation int

// Granularity is the unit a rule judges. The levels form a scale: each is a
// refinement of the one above, so the number of verdicts a file yields never
// decreases as the level gets finer.
type Granularity int

// Fixture is one scenario directory exercising a rule. File is the file its
// expectation applies to.
type Fixture struct {
	Name   string
	File   string
	Expect Expectation
}

// Prompt is a parsed prompt.md: its frontmatter settings and its body.
type Prompt struct {
	IncludeSource bool
	Granularity   Granularity
	Targets       []string
	Include       []string
	Body          string
}

const (
	ExpectPass Expectation = iota + 1
	ExpectFail
)

const (
	// GranularityFile judges the file as a single unit, keyed by its path.
	GranularityFile Granularity = iota + 1
	// GranularityFunction judges each thing the file runs or declares under its own
	// name. Which of them count is the included fragments' answer, not this value's:
	// with the tests fragment it is each test, with none it is each declaration.
	GranularityFunction
	// GranularityTestCase judges each independently nameable leaf: one case of a
	// test, or the test itself when it declares no cases.
	GranularityTestCase
)

// Load reads <rulesDir>/<name>/prompt.md. The kinds of file a rule may target are
// passed in rather than compiled in like the fragments: a repository's config
// extends that vocabulary, so it is only known once aritu.yml has been read.
func Load(rulesDir, name string, knownTargets []string) (Rule, error) {
	dir := filepath.Join(rulesDir, name)
	raw, err := os.ReadFile(filepath.Join(dir, promptFileName))
	if err != nil {
		return Rule{}, fmt.Errorf("rule %q: %w", name, err)
	}
	prompt, err := ParsePrompt(string(raw), knownTargets)
	if err != nil {
		return Rule{}, fmt.Errorf("rule %q: %w", name, err)
	}
	return Rule{
		Name:          name,
		Dir:           dir,
		Prompt:        prompt.Body,
		Targets:       prompt.Targets,
		Include:       prompt.Include,
		IncludeSource: prompt.IncludeSource,
		Granularity:   prompt.Granularity,
	}, nil
}

// List names every rule in the directory, sorted. Only directories count, and a
// name starting with _ is parked: it stays on disk, keeps its fixtures and still
// loads when somebody names it, but no sweep picks it up on its own.
//
// Parking is what a repository does with a rule it is not ready to enforce, and it
// beats the alternatives — deleting the rule loses the prompt that took the work,
// and listing the others in aritu.yml means editing that list every time a rule is
// added.
func List(rulesDir string) ([]string, error) {
	entries, err := os.ReadDir(rulesDir)
	if err != nil {
		return nil, fmt.Errorf("rules directory: %w", err)
	}
	names := make([]string, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() && !IsParked(entry.Name()) {
			names = append(names, entry.Name())
		}
	}
	if len(names) == 0 {
		return nil, fmt.Errorf("rules directory %s holds no rules", rulesDir)
	}
	slices.Sort(names)
	return names, nil
}

// IsParked answers whether a rule name is one a sweep leaves alone. Naming it
// explicitly still runs it, the way a pattern naming a file inside the rules
// directory still judges it: what was asked for outranks what was derived.
func IsParked(name string) bool {
	return strings.HasPrefix(name, parkedPrefix)
}

// LoadFixtures lists the rule's fixture directories, sorted by name.
func LoadFixtures(r Rule) ([]Fixture, error) {
	dir := filepath.Join(r.Dir, fixturesDirName)
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("rule %q: %w", r.Name, err)
	}
	fixtures := make([]Fixture, 0, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		fixture, err := loadFixture(dir, entry.Name())
		if err != nil {
			return nil, fmt.Errorf("rule %q: %w", r.Name, err)
		}
		fixtures = append(fixtures, fixture)
	}
	if len(fixtures) == 0 {
		return nil, fmt.Errorf("rule %q: no fixture directories in %s", r.Name, dir)
	}
	slices.SortFunc(fixtures, func(a, b Fixture) int { return strings.Compare(a.Name, b.Name) })
	return fixtures, nil
}

// ParsePrompt splits a prompt.md into frontmatter and body. A missing granularity
// or targets key is an error: defaulting either silently would change which files
// reach the model, or what it is asked about them, without anyone noticing.
//
// include_source defaults to false, because false is what a rule needs unless it
// is about tests: sending the implementation is only meaningful where there is a
// file under test to find, and most rules judge a file on its own terms. The key
// is written out only where a rule wants the pairing.
func ParsePrompt(raw string, knownTargets []string) (Prompt, error) {
	lines := strings.Split(raw, "\n")
	if len(lines) == 0 || !isFrontmatterDelimiter(lines[0]) {
		return Prompt{}, errors.New("prompt.md: missing frontmatter, first line must be ---")
	}
	closing := indexOfDelimiter(lines[1:])
	if closing < 0 {
		return Prompt{}, errors.New("prompt.md: unterminated frontmatter, no closing ---")
	}
	closing++
	var front frontmatter
	if err := yaml.Unmarshal([]byte(strings.Join(lines[1:closing], "\n")), &front); err != nil {
		return Prompt{}, fmt.Errorf("prompt.md: malformed frontmatter: %w", err)
	}
	if front.Granularity == nil {
		return Prompt{}, errors.New("prompt.md: frontmatter must set granularity")
	}
	granularity, err := ParseGranularity(*front.Granularity)
	if err != nil {
		return Prompt{}, fmt.Errorf("prompt.md: %w", err)
	}
	if err := checkTargetsAreKnown(front.Targets, knownTargets); err != nil {
		return Prompt{}, fmt.Errorf("prompt.md: %w", err)
	}
	if err := checkIncludesAreKnown(front.Include); err != nil {
		return Prompt{}, fmt.Errorf("prompt.md: %w", err)
	}
	return Prompt{
		IncludeSource: front.IncludeSource,
		Granularity:   granularity,
		Targets:       front.Targets,
		Include:       front.Include,
		Body:          joinAfterLeadingBlanks(lines[closing+1:]),
	}, nil
}

// ParseGranularity reads the unit a rule declares it judges.
func ParseGranularity(name string) (Granularity, error) {
	granularity, isKnown := granularityNames[name]
	if !isKnown {
		return 0, fmt.Errorf("granularity %q: must be file, function or test_case", name)
	}
	return granularity, nil
}

// checkTargetsAreKnown rejects a rule that is about no kind of file, or about one
// this repository has no answer for. Neither can be defaulted or skipped: a rule
// targeting nothing is handed no file, and a misspelled kind matches none, and both
// of those run nothing and report green.
func checkTargetsAreKnown(targets, known []string) error {
	if len(targets) == 0 {
		return fmt.Errorf("frontmatter must set targets: one or more of %s", strings.Join(known, ", "))
	}
	for _, name := range targets {
		if !slices.Contains(known, name) {
			return fmt.Errorf("target %q: must be one of %s", name, strings.Join(known, ", "))
		}
	}
	return nil
}

// checkIncludesAreKnown rejects a fragment this binary does not carry. Rules are
// read from a repository and prompts are compiled in, so an include naming a
// fragment that was renamed or never existed has to fail when the rule is loaded
// rather than reach a model as a gap in the prompt.
func checkIncludesAreKnown(includes []string) error {
	for _, name := range includes {
		if !prompts.IsKnown(name) {
			return fmt.Errorf("include %q: must be one of %s", name, strings.Join(prompts.Known(), ", "))
		}
	}
	return nil
}

// ParseExpectation reads the pass-/fail- prefix a fixture directory carries.
func ParseExpectation(dirName string) (Expectation, error) {
	for prefix, expect := range expectationPrefixes {
		if strings.HasPrefix(dirName, prefix) {
			return expect, nil
		}
	}
	return 0, fmt.Errorf("fixture %q: name must start with pass- or fail-", dirName)
}

// FindSource locates the implementation a test file covers: the first candidate
// its naming convention offers that is actually a file on disk.
//
// Resolution has to touch the filesystem because a mirrored source tree and a file
// beside the test are both plausible layouts, and no reading of the path alone
// decides between them.
//
// The failure names every path it looked at. Four of seven rules need the source,
// and a rule that skips a file is only useful if the reader can see where aritu
// searched and add the file — or the layout — that was missing.
func FindSource(testPath string) (string, error) {
	candidates := testpath.SourceCandidates(testPath)
	if len(candidates) == 0 {
		return "", fmt.Errorf("%s matches no test file naming convention aritu knows", testPath)
	}
	for _, candidate := range candidates {
		info, err := os.Stat(candidate)
		if err == nil && !info.IsDir() {
			return candidate, nil
		}
	}
	return "", fmt.Errorf("no implementation found for %s, looked for %s", testPath, strings.Join(candidates, ", "))
}

// String renders an expectation as "pass" or "fail".
func (e Expectation) String() string {
	switch e {
	case ExpectPass:
		return "pass"
	case ExpectFail:
		return "fail"
	default:
		panic(fmt.Sprintf("unknown expectation: %d", int(e)))
	}
}

// String renders a granularity as it is written in frontmatter.
func (g Granularity) String() string {
	switch g {
	case GranularityFile:
		return "file"
	case GranularityFunction:
		return "function"
	case GranularityTestCase:
		return "test_case"
	default:
		panic(fmt.Sprintf("unknown granularity: %d", int(g)))
	}
}

const (
	promptFileName  = "prompt.md"
	fixturesDirName = "fixtures"
	parkedPrefix    = "_"
)

var expectationPrefixes = map[string]Expectation{
	"pass-": ExpectPass,
	"fail-": ExpectFail,
}

var granularityNames = map[string]Granularity{
	"file":      GranularityFile,
	"function":  GranularityFunction,
	"test_case": GranularityTestCase,
}

func loadFixture(fixturesDir, name string) (Fixture, error) {
	expect, err := ParseExpectation(name)
	if err != nil {
		return Fixture{}, err
	}
	dir := filepath.Join(fixturesDir, name)
	file, err := findFixtureFile(dir)
	if err != nil {
		return Fixture{}, err
	}
	return Fixture{Name: name, File: file, Expect: expect}, nil
}

// findFixtureFile picks the file a fixture's expectation applies to, whatever the
// language. A test file wins when the directory holds exactly one — a rule that
// pairs a test with its implementation keeps both in the fixture, and the test is
// the judged half — and a directory with no test file offers exactly one plain
// source file instead, which is what a fixture for a rule about ordinary code
// holds. Either way exactly one file qualifies: two would leave which of them the
// expectation applies to undecided.
func findFixtureFile(dir string) (string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return "", err
	}
	tests := make([]string, 0, 1)
	sources := make([]string, 0, 1)
	for _, entry := range entries {
		if entry.IsDir() || !isSourceFile(entry.Name()) {
			continue
		}
		if testpath.IsTestFile(entry.Name()) {
			tests = append(tests, filepath.Join(dir, entry.Name()))
			continue
		}
		sources = append(sources, filepath.Join(dir, entry.Name()))
	}
	if len(tests) == 1 {
		return tests[0], nil
	}
	if len(tests) == 0 && len(sources) == 1 {
		return sources[0], nil
	}
	return "", fmt.Errorf("fixture %s: want exactly one test file, or exactly one source file and no test file, found %d test and %d source files", dir, len(tests), len(sources))
}

func isSourceFile(name string) bool {
	return slices.Contains(testpath.Extensions(), filepath.Ext(name))
}

type frontmatter struct {
	IncludeSource bool     `yaml:"include_source"`
	Granularity   *string  `yaml:"granularity"`
	Targets       []string `yaml:"targets"`
	Include       []string `yaml:"include"`
}

func isFrontmatterDelimiter(line string) bool {
	return strings.TrimSpace(line) == "---"
}

func indexOfDelimiter(lines []string) int {
	for i, line := range lines {
		if isFrontmatterDelimiter(line) {
			return i
		}
	}
	return -1
}

func joinAfterLeadingBlanks(lines []string) string {
	for i, line := range lines {
		if strings.TrimSpace(line) != "" {
			return strings.Join(lines[i:], "\n")
		}
	}
	return ""
}
