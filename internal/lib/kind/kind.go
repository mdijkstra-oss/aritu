package kind

import (
	"fmt"
	"maps"
	"path/filepath"
	"slices"

	"github.com/matthijn/aritu/internal/lib/glob"
	"github.com/matthijn/aritu/internal/lib/language"
	"github.com/matthijn/aritu/internal/lib/testpath"
)

type Kind struct {
	Name     string
	Patterns []string
	Covers   func(path string) bool
}

type Set map[string]Kind

func Resolve(base string, declared map[string][]string) (Set, error) {
	resolved := make(Set, len(builtins)+len(declared))
	for _, builtin := range builtins {
		resolved[builtin.Name] = Kind{
			Name:     builtin.Name,
			Patterns: allRooted(base, builtin.Patterns),
			Covers:   builtin.Covers,
		}
	}
	for name, patterns := range declared {
		if err := checkPatternsAreUsable(name, patterns); err != nil {
			return nil, err
		}
		resolved[name] = declaredKind(name, patterns)
	}
	return resolved, nil
}

func (s Set) Names() []string {
	return slices.Sorted(maps.Keys(s))
}

func (s Set) Covers(names []string, path string) bool {
	for _, name := range names {
		if s.named(name).Covers(path) {
			return true
		}
	}
	return false
}

func (s Set) Expand(names []string) ([]string, error) {
	candidates, err := glob.ExpandGenerated(s.patternsOf(names))
	if err != nil {
		return nil, err
	}
	return s.filterCovered(names, candidates), nil
}

var builtins = []Kind{
	{Name: "tests", Patterns: sourcePatterns, Covers: testpath.IsTestFile},
	{Name: "code", Patterns: sourcePatterns, Covers: language.IsSourceFile},
	{Name: "docs", Patterns: docPatterns, Covers: isDocFile},
}

var (
	docExtensions = []string{".md", ".mdx", ".txt"}

	sourcePatterns = patternsFor(language.Extensions())
	docPatterns    = patternsFor(docExtensions)
)

func declaredKind(name string, patterns []string) Kind {
	return Kind{Name: name, Patterns: patterns, Covers: coveredByAny(patterns)}
}

func coveredByAny(patterns []string) func(string) bool {
	return func(path string) bool { return glob.MatchesAny(patterns, path) }
}

func checkPatternsAreUsable(name string, patterns []string) error {
	if len(patterns) == 0 {
		return fmt.Errorf("target %q: names no patterns, so a rule about it would judge nothing", name)
	}
	if err := glob.CheckAll(patterns); err != nil {
		return fmt.Errorf("target %q: %w", name, err)
	}
	return nil
}

func patternsFor(extensions []string) []string {
	patterns := make([]string, 0, len(extensions))
	for _, extension := range extensions {
		patterns = append(patterns, "**/*"+extension)
	}
	return patterns
}

func isDocFile(path string) bool {
	return slices.Contains(docExtensions, filepath.Ext(path))
}

func allRooted(base string, patterns []string) []string {
	rooted := make([]string, 0, len(patterns))
	for _, pattern := range patterns {
		rooted = append(rooted, glob.Rooted(base, pattern))
	}
	return rooted
}

func (s Set) filterCovered(names, paths []string) []string {
	covered := make([]string, 0, len(paths))
	for _, path := range paths {
		if s.Covers(names, path) {
			covered = append(covered, path)
		}
	}
	return covered
}

func (s Set) patternsOf(names []string) []string {
	patterns := make([]string, 0, len(names))
	for _, name := range names {
		for _, pattern := range s.named(name).Patterns {
			if !slices.Contains(patterns, pattern) {
				patterns = append(patterns, pattern)
			}
		}
	}
	return patterns
}

// named panics because a rule's targets are checked against this set when the rule
// is loaded: a name arriving here is one that reached matching without being
// validated, which no repository can cause.
func (s Set) named(name string) Kind {
	found, isKnown := s[name]
	if !isKnown {
		panic(fmt.Sprintf("target %q reached matching without being validated", name))
	}
	return found
}
