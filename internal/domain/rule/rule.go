package rule

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/matthijn/aritu/internal/lib/language"
	"github.com/matthijn/aritu/internal/lib/testpath"
)

type Rule struct {
	Name          string
	Dir           string
	Prompt        string
	Targets       []string
	IncludeSource bool
	Granularity   Granularity
	Priority      Priority
}

type Expectation int

type Granularity int

type Priority int

type Fixture struct {
	Name   string
	File   string
	Expect Expectation
}

type Prompt struct {
	IncludeSource bool
	Granularity   Granularity
	Priority      Priority
	Targets       []string
	Body          string
}

const (
	ExpectPass Expectation = iota + 1
	ExpectFail
)

const (
	GranularityFile Granularity = iota + 1
	GranularityFunction
	GranularityTestCase
	GranularityComment
	GranularityDeclaration
)

const (
	PriorityUndeclared Priority = iota
	PriorityMed
	PriorityHigh
	PrioritySevere
)

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
		IncludeSource: prompt.IncludeSource,
		Granularity:   prompt.Granularity,
		Priority:      prompt.Priority,
	}, nil
}

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

func IsParked(name string) bool {
	return strings.HasPrefix(name, parkedPrefix)
}

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
	priority, err := priorityOr(front.Priority)
	if err != nil {
		return Prompt{}, fmt.Errorf("prompt.md: %w", err)
	}
	if err := checkTargetsAreKnown(front.Targets, knownTargets); err != nil {
		return Prompt{}, fmt.Errorf("prompt.md: %w", err)
	}
	return Prompt{
		IncludeSource: front.IncludeSource,
		Granularity:   granularity,
		Priority:      priority,
		Targets:       front.Targets,
		Body:          joinAfterLeadingBlanks(lines[closing+1:]),
	}, nil
}

func ParseGranularity(name string) (Granularity, error) {
	granularity, isKnown := granularityNames[name]
	if !isKnown {
		return 0, fmt.Errorf("granularity %q: must be file, function, test_case, comment or declaration", name)
	}
	return granularity, nil
}

func ParsePriority(name string) (Priority, error) {
	priority, isKnown := priorityNames[name]
	if !isKnown {
		return 0, fmt.Errorf("priority %q: must be med, high or severe", name)
	}
	return priority, nil
}

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

func ParseExpectation(dirName string) (Expectation, error) {
	for prefix, expect := range expectationPrefixes {
		if strings.HasPrefix(dirName, prefix) {
			return expect, nil
		}
	}
	return 0, fmt.Errorf("fixture %q: name must start with pass- or fail-", dirName)
}

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

func (g Granularity) String() string {
	switch g {
	case GranularityFile:
		return "file"
	case GranularityFunction:
		return "function"
	case GranularityTestCase:
		return "test_case"
	case GranularityComment:
		return "comment"
	case GranularityDeclaration:
		return "declaration"
	default:
		panic(fmt.Sprintf("unknown granularity: %d", int(g)))
	}
}

func (p Priority) Band() Priority {
	if p == PriorityUndeclared {
		return PriorityMed
	}
	return p
}

func (p Priority) String() string {
	switch p.Band() {
	case PriorityMed:
		return "med"
	case PriorityHigh:
		return "high"
	case PrioritySevere:
		return "severe"
	default:
		panic(fmt.Sprintf("unknown priority: %d", int(p)))
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
	"file":        GranularityFile,
	"function":    GranularityFunction,
	"test_case":   GranularityTestCase,
	"comment":     GranularityComment,
	"declaration": GranularityDeclaration,
}

var priorityNames = map[string]Priority{
	"med":    PriorityMed,
	"high":   PriorityHigh,
	"severe": PrioritySevere,
}

func priorityOr(declared *string) (Priority, error) {
	if declared == nil {
		return PriorityMed, nil
	}
	return ParsePriority(*declared)
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

func findFixtureFile(dir string) (string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return "", err
	}
	tests := make([]string, 0, 1)
	sources := make([]string, 0, 1)
	for _, entry := range entries {
		if entry.IsDir() || !language.IsSourceFile(entry.Name()) {
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


type frontmatter struct {
	IncludeSource bool     `yaml:"include_source"`
	Granularity   *string  `yaml:"granularity"`
	Priority      *string  `yaml:"priority"`
	Targets       []string `yaml:"targets"`
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
