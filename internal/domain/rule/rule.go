package rule

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"gopkg.in/yaml.v3"
)

// Rule is one linting rule loaded from a directory under the rules dir.
type Rule struct {
	Name          string
	Dir           string
	Prompt        string
	IncludeSource bool
}

// Expectation is the pass/fail outcome a fixture directory name asserts.
type Expectation int

// Fixture is one scenario directory exercising a rule.
type Fixture struct {
	Name     string
	TestFile string
	Expect   Expectation
}

// Prompt is a parsed prompt.md: its frontmatter settings and its body.
type Prompt struct {
	IncludeSource bool
	Body          string
}

const (
	ExpectPass Expectation = iota + 1
	ExpectFail
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
		IncludeSource: prompt.IncludeSource,
	}, nil
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
	return Prompt{
		IncludeSource: *front.IncludeSource,
		Body:          joinAfterLeadingBlanks(lines[closing+1:]),
	}, nil
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

// SourcePathFor maps a Go test file to the implementation it covers, following
// the file_test.go -> file.go convention. It reports false when the path is not
// a test file.
func SourcePathFor(testPath string) (string, bool) {
	if !strings.HasSuffix(testPath, testSuffix) {
		return "", false
	}
	if strings.TrimSuffix(filepath.Base(testPath), testSuffix) == "" {
		return "", false
	}
	return strings.TrimSuffix(testPath, testSuffix) + ".go", true
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

const (
	promptFileName  = "prompt.md"
	fixturesDirName = "fixtures"
	testSuffix      = "_test.go"
)

var expectationPrefixes = map[string]Expectation{
	"pass-": ExpectPass,
	"fail-": ExpectFail,
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

func findTestFile(dir string) (string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return "", err
	}
	found := make([]string, 0, 1)
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), testSuffix) {
			continue
		}
		found = append(found, filepath.Join(dir, entry.Name()))
	}
	if len(found) != 1 {
		return "", fmt.Errorf("fixture %s: want exactly one *_test.go file, found %d", dir, len(found))
	}
	return found[0], nil
}

type frontmatter struct {
	IncludeSource *bool `yaml:"include_source"`
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
