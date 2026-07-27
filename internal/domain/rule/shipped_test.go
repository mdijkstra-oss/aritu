package rule_test

import (
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
var shippedRules = []struct {
	name          string
	targets       []string
	granularity   rule.Granularity
	includeSource bool
	include       []string
}{
	{name: "no-gaps", targets: []string{"tests"}, granularity: rule.GranularityFile, includeSource: true, include: []string{"tests"}},
	{name: "no-redundancy", targets: []string{"tests"}, granularity: rule.GranularityFile, includeSource: true, include: []string{"tests"}},
	{name: "proves-what-it-claims", targets: []string{"tests"}, granularity: rule.GranularityTestCase, includeSource: true, include: []string{"tests"}},
	{name: "readable", targets: []string{"tests"}, granularity: rule.GranularityFunction, includeSource: false, include: []string{"tests"}},
	{name: "self-contained", targets: []string{"tests"}, granularity: rule.GranularityFile, includeSource: false, include: []string{"tests"}},
	{name: "tests-behavior-not-implementation", targets: []string{"tests"}, granularity: rule.GranularityFunction, includeSource: true, include: []string{"tests"}},
	{name: "tests-one-thing", targets: []string{"tests"}, granularity: rule.GranularityFunction, includeSource: false, include: []string{"tests"}},
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
	names, err := rule.List(shippedRulesDir)
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}

	want := make([]string, 0, len(shippedRules))
	for _, shipped := range shippedRules {
		want = append(want, shipped.name)
	}
	if !slices.Equal(names, want) {
		t.Errorf("rules directory holds %v, want exactly %v", names, want)
	}
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
