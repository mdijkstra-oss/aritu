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

// Fixture is one scenario directory exercising a rule.
type Fixture struct {
	Name     string
	TestFile string
	Expect   Expectation
}

// Prompt is a parsed prompt.md: its frontmatter settings and its body.
type Prompt struct {
	IncludeSource bool
	Granularity   Granularity
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

// Load reads <rulesDir>/<name>/prompt.md.
func Load(rulesDir, name string) (Rule, error) {
	dir := filepath.Join(rulesDir, name)
	raw, err := os.ReadFile(filepath.Join(dir, promptFileName))
	if err != nil {
		return Rule{}, fmt.Errorf("rule %q: %w", name, err)
	}
	prompt, err := ParsePrompt(string(raw))
	if err != nil {
		return Rule{}, fmt.Errorf("rule %q: %w", name, err)
	}
	return Rule{
		Name:          name,
		Dir:           dir,
		Prompt:        prompt.Body,
		Include:       prompt.Include,
		IncludeSource: prompt.IncludeSource,
		Granularity:   prompt.Granularity,
	}, nil
}

// List names every rule in the directory, sorted. Only directories count.
func List(rulesDir string) ([]string, error) {
	entries, err := os.ReadDir(rulesDir)
	if err != nil {
		return nil, fmt.Errorf("rules directory: %w", err)
	}
	names := make([]string, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() {
			names = append(names, entry.Name())
		}
	}
	if len(names) == 0 {
		return nil, fmt.Errorf("rules directory %s holds no rules", rulesDir)
	}
	slices.Sort(names)
	return names, nil
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

// ParsePrompt splits a prompt.md into frontmatter and body. A missing
// include_source key is an error: defaulting it silently would change which
// files reach the model without anyone noticing.
func ParsePrompt(raw string) (Prompt, error) {
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
	if front.IncludeSource == nil {
		return Prompt{}, errors.New("prompt.md: frontmatter must set include_source")
	}
	if front.Granularity == nil {
		return Prompt{}, errors.New("prompt.md: frontmatter must set granularity")
	}
	granularity, err := ParseGranularity(*front.Granularity)
	if err != nil {
		return Prompt{}, fmt.Errorf("prompt.md: %w", err)
	}
	if err := checkIncludesAreKnown(front.Include); err != nil {
		return Prompt{}, fmt.Errorf("prompt.md: %w", err)
	}
	return Prompt{
		IncludeSource: *front.IncludeSource,
		Granularity:   granularity,
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
	testFile, err := findTestFile(dir)
	if err != nil {
		return Fixture{}, err
	}
	return Fixture{Name: name, TestFile: testFile, Expect: expect}, nil
}

// findTestFile insists on exactly one test file per fixture directory, whatever
// the language. One is what makes a fixture unambiguous: two would leave which
// file the expectation applies to undecided.
func findTestFile(dir string) (string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return "", err
	}
	found := make([]string, 0, 1)
	for _, entry := range entries {
		if entry.IsDir() || !testpath.IsTestFile(entry.Name()) {
			continue
		}
		found = append(found, filepath.Join(dir, entry.Name()))
	}
	if len(found) != 1 {
		return "", fmt.Errorf("fixture %s: want exactly one test file, found %d", dir, len(found))
	}
	return found[0], nil
}

type frontmatter struct {
	IncludeSource *bool    `yaml:"include_source"`
	Granularity   *string  `yaml:"granularity"`
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
