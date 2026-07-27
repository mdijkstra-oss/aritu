package rule_test

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/matthijn/aritu/internal/domain/rule"
)

// shippedRulesDir is the rule set aritu ships, read from the repository rather
// than rebuilt in a temp directory. These tests are about that corpus: a prompt
// that quietly acquires a language, a rule that loses a fixture language, and a
// granularity flipped by hand are all invisible to a test over synthetic input.
var shippedRulesDir = filepath.Join("..", "..", "..", "rules")

// shippedRules is the set the feature defines, with the files each rule is about,
// the pairing it is judged at and the fragments its prompt carries. Every key is
// load-bearing: two properties that disagree about any of them cannot share a
// verdict call, so a flipped value silently changes which files reach the model,
// what it is asked, and what it can see.
//
// Every name is parked. The seven are aritu's own material rather than the rule
// set this repository enforces on itself, so they sit out of every sweep and are
// reached by being named.
var shippedRules = []struct {
	name          string
	targets       []string
	granularity   rule.Granularity
	includeSource bool
	include       []string
}{
	{name: "_no-gaps", targets: []string{"tests"}, granularity: rule.GranularityFile, includeSource: true, include: []string{"tests"}},
	{name: "_no-redundancy", targets: []string{"tests"}, granularity: rule.GranularityFile, includeSource: true, include: []string{"tests"}},
	{name: "_proves-what-it-claims", targets: []string{"tests"}, granularity: rule.GranularityTestCase, includeSource: true, include: []string{"tests"}},
	{name: "_readable", targets: []string{"tests"}, granularity: rule.GranularityFunction, includeSource: false, include: []string{"tests"}},
	{name: "_self-contained", targets: []string{"tests"}, granularity: rule.GranularityFile, includeSource: false, include: []string{"tests"}},
	{name: "_tests-behavior-not-implementation", targets: []string{"tests"}, granularity: rule.GranularityFunction, includeSource: true, include: []string{"tests"}},
	{name: "_tests-one-thing", targets: []string{"tests"}, granularity: rule.GranularityFunction, includeSource: false, include: []string{"tests"}},
}

// knownTargets is what a repository running the shipped set would have resolved
// before reading any of it: the built-in kinds, which are all these rules name.
var knownTargets = []string{"code", "docs", "tests"}

// requiredFixtureLanguages is the coverage floor. A prompt that passes its fixtures
// in one language and nothing else is a prompt describing that language's shapes,
// and nothing but a fixture in another ecosystem detects it.
var requiredFixtureLanguages = []string{"Go", "TypeScript"}

var fixtureLanguages = map[string]string{
	".go":  "Go",
	".ts":  "TypeScript",
	".tsx": "TypeScript",
}

func TestTheShippedRuleSetIsTheSevenGroupedRules(t *testing.T) {
	want := make([]string, 0, len(shippedRules))
	for _, shipped := range shippedRules {
		want = append(want, shipped.name)
	}

	got := groupedRuleNames(t)

	if !slices.Equal(got, want) {
		t.Errorf("the grouped rules are %v, want exactly %v", got, want)
	}
}

// groupedRuleNames reads the directories List deliberately leaves out, which is
// the only way to see the shipped set now that all seven of them are parked.
//
// Two populations are parked here and only one of them is aritu's: the grouped
// rules sit under a single _, and the rules this repository is still writing
// prompts for sit under __. aritu draws no distinction — anything with a leading _
// is parked and that is all it knows — so the depth is this repository's own
// bookkeeping, and the shipped set is the shallow one.
func groupedRuleNames(t *testing.T) []string {
	t.Helper()
	entries, err := os.ReadDir(shippedRulesDir)
	if err != nil {
		t.Fatalf("reading %s: %v", shippedRulesDir, err)
	}
	names := make([]string, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() && isGroupedRule(entry.Name()) {
			names = append(names, entry.Name())
		}
	}
	slices.Sort(names)
	return names
}

func isGroupedRule(name string) bool {
	return rule.IsParked(name) && !strings.HasPrefix(name, "__")
}

func TestEveryShippedRuleDeclaresTheLevelAndEvidenceItsPropertyNeeds(t *testing.T) {
	for _, shipped := range shippedRules {
		t.Run(shipped.name, func(t *testing.T) {
			loaded, err := rule.Load(shippedRulesDir, shipped.name, knownTargets)
			if err != nil {
				t.Fatalf("Load() error = %v", err)
			}

			if !slices.Equal(loaded.Targets, shipped.targets) {
				t.Errorf("targets = %v, want %v", loaded.Targets, shipped.targets)
			}
			if loaded.Granularity != shipped.granularity {
				t.Errorf("granularity = %s, want %s", loaded.Granularity, shipped.granularity)
			}
			if loaded.IncludeSource != shipped.includeSource {
				t.Errorf("include_source = %v, want %v", loaded.IncludeSource, shipped.includeSource)
			}
			if !slices.Equal(loaded.Include, shipped.include) {
				t.Errorf("include = %v, want %v", loaded.Include, shipped.include)
			}
			if strings.TrimSpace(loaded.Prompt) == "" {
				t.Error("prompt body is empty, so the rule states no criterion")
			}
			if strings.TrimSpace(loaded.Description) == "" {
				t.Error("description is empty, so the rule takes a heading in the rulebook and asks for nothing under it")
			}
		})
	}
}

func TestEveryShippedRuleProvesItselfInEveryLanguage(t *testing.T) {
	for _, shipped := range shippedRules {
		t.Run(shipped.name, func(t *testing.T) {
			loaded, err := rule.Load(shippedRulesDir, shipped.name, knownTargets)
			if err != nil {
				t.Fatalf("Load() error = %v", err)
			}
			fixtures, err := rule.LoadFixtures(loaded)
			if err != nil {
				t.Fatalf("LoadFixtures() error = %v", err)
			}

			covered := map[rule.Expectation][]string{}
			for _, fixture := range fixtures {
				language, isKnown := fixtureLanguages[filepath.Ext(fixture.TestFile)]
				if !isKnown {
					t.Errorf("fixture %s is written in no language the coverage floor counts", fixture.Name)
					continue
				}
				covered[fixture.Expect] = append(covered[fixture.Expect], language)
			}

			for _, expect := range []rule.Expectation{rule.ExpectPass, rule.ExpectFail} {
				for _, language := range requiredFixtureLanguages {
					if !slices.Contains(covered[expect], language) {
						t.Errorf("no %s fixture in %s", expect, language)
					}
				}
			}
		})
	}
}
