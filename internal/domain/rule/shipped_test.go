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

// shippedRules is the set the feature defines, with the pairing each rule is
// judged at. Both keys are load-bearing: two properties that disagree about
// either cannot share a verdict call, so a flipped value silently changes what
// the model is asked and what it can see.
var shippedRules = []struct {
	name          string
	granularity   rule.Granularity
	includeSource bool
}{
	{name: "no-gaps", granularity: rule.GranularityFile, includeSource: true},
	{name: "no-redundancy", granularity: rule.GranularityFile, includeSource: true},
	{name: "proves-what-it-claims", granularity: rule.GranularityTest, includeSource: true},
	{name: "readable", granularity: rule.GranularityFunction, includeSource: false},
	{name: "self-contained", granularity: rule.GranularityFile, includeSource: false},
	{name: "tests-behavior-not-implementation", granularity: rule.GranularityFunction, includeSource: true},
	{name: "tests-one-thing", granularity: rule.GranularityFunction, includeSource: false},
}

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
			loaded, err := rule.Load(shippedRulesDir, shipped.name)
			if err != nil {
				t.Fatalf("Load() error = %v", err)
			}

			if loaded.Granularity != shipped.granularity {
				t.Errorf("granularity = %s, want %s", loaded.Granularity, shipped.granularity)
			}
			if loaded.IncludeSource != shipped.includeSource {
				t.Errorf("include_source = %v, want %v", loaded.IncludeSource, shipped.includeSource)
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
			loaded, err := rule.Load(shippedRulesDir, shipped.name)
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
