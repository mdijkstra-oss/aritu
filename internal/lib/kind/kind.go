// Package kind answers which files a rule is about.
//
// Scoping a rule to files is two questions with two homes. Which files a rule makes
// sense for is intrinsic to the rule, in every repository, forever. Which files in
// this repository are of that sort is intrinsic to the repository. So a rule names
// a kind, and this package maps kinds to files.
//
// A kind carries both a pattern list and a membership predicate because they answer
// different halves. Patterns generate the candidates a sweep starts from, since a
// predicate cannot enumerate a filesystem. The predicate decides membership exactly,
// since no glob expresses what a test file is called across four ecosystems without
// becoming a second copy of the table that already knows.
package kind

import (
	"fmt"
	"maps"
	"path/filepath"
	"slices"

	"github.com/matthijn/aritu/internal/lib/glob"
	"github.com/matthijn/aritu/internal/lib/testpath"
)

// Kind is one named sort of file, held as data so that supporting another sort is a
// row in a table rather than a branch in the code.
type Kind struct {
	Name     string
	Patterns []string
	Covers   func(path string) bool
}

// Set is the vocabulary one repository judges by: the built-ins overlaid with
// whatever its config declared. The keys are open, so a rule may name a sort of
// file this binary never heard of and the repository that wrote the rule can say
// what it means.
type Set map[string]Kind

// Resolve builds that vocabulary.
//
// A built-in generates against base, because which files in this repository are
// tests is a question about this repository. Its membership stays rootless, because
// what a test file is called is not: a path named on the command line is judged the
// same wherever it sits.
//
// A declared key replaces a built-in wholesale, patterns and membership together. A
// repository overriding tests is saying it knows better than the conventions table,
// and quietly keeping that table as a refinement over its patterns would make the
// override lie.
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

// Names lists every kind a rule may target, sorted.
func (s Set) Names() []string {
	return slices.Sorted(maps.Keys(s))
}

// Covers reports whether a path is of any of the named kinds.
func (s Set) Covers(names []string, path string) bool {
	for _, name := range names {
		if s.named(name).Covers(path) {
			return true
		}
	}
	return false
}

// Expand lists every existing file the named kinds cover, sorted and deduplicated.
// The patterns only generate candidates, so what they over-select is filtered by
// the same membership test that pairs a file with a rule.
func (s Set) Expand(names []string) ([]string, error) {
	candidates, err := glob.ExpandGenerated(s.patternsOf(names))
	if err != nil {
		return nil, err
	}
	return s.filterCovered(names, candidates), nil
}

// builtins is what aritu knows about kinds of file without being told. tests and
// code take their patterns from the one extension list the conventions table is
// indexed by, so an ecosystem added there widens both with no second edit.
//
// code deliberately overlaps tests: a test file has comments like any other source
// file, and these are named matchers rather than a partition of the tree. A
// repository wanting source without tests writes that itself.
var builtins = []Kind{
	{Name: "tests", Patterns: sourcePatterns, Covers: testpath.IsTestFile},
	{Name: "code", Patterns: sourcePatterns, Covers: isSourceFile},
	{Name: "docs", Patterns: docPatterns, Covers: isDocFile},
}

var (
	sourceExtensions = testpath.Extensions()
	docExtensions    = []string{".md", ".mdx"}

	sourcePatterns = patternsFor(sourceExtensions)
	docPatterns    = patternsFor(docExtensions)
)

// declaredKind is a repository's own answer, where the patterns both generate and
// decide: a repository that has named the files it means has said exactly which
// ones they are.
func declaredKind(name string, patterns []string) Kind {
	return Kind{Name: name, Patterns: patterns, Covers: coveredByAny(patterns)}
}

func coveredByAny(patterns []string) func(string) bool {
	return func(path string) bool {
		return slices.ContainsFunc(patterns, func(pattern string) bool { return matches(pattern, path) })
	}
}

// checkPatternsAreUsable refuses a kind that could never hold a file. Both failures
// report green if they are let through: a rule about a kind matching nothing runs
// over nothing and passes.
func checkPatternsAreUsable(name string, patterns []string) error {
	if len(patterns) == 0 {
		return fmt.Errorf("target %q: names no patterns, so a rule about it would judge nothing", name)
	}
	for _, pattern := range patterns {
		if !glob.IsValid(pattern) {
			return fmt.Errorf("target %q: pattern %q: syntax error", name, pattern)
		}
	}
	return nil
}

// patternsFor generates one pattern per extension, at any depth, because a kind
// names a sort of file rather than a place in the tree.
func patternsFor(extensions []string) []string {
	patterns := make([]string, 0, len(extensions))
	for _, extension := range extensions {
		patterns = append(patterns, "**/*"+extension)
	}
	return patterns
}

func isSourceFile(path string) bool { return hasExtensionIn(sourceExtensions, path) }

func isDocFile(path string) bool { return hasExtensionIn(docExtensions, path) }

func hasExtensionIn(extensions []string, path string) bool {
	return slices.Contains(extensions, filepath.Ext(path))
}

func allRooted(base string, patterns []string) []string {
	rooted := make([]string, 0, len(patterns))
	for _, pattern := range patterns {
		rooted = append(rooted, glob.Rooted(base, pattern))
	}
	return rooted
}

// filterCovered keeps the paths that are of one of the named kinds, in order.
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

// matches panics for the same reason: every declared pattern was checked when the
// set was resolved, and a built-in one is written in this file.
func matches(pattern, path string) bool {
	covered, err := glob.Match(pattern, path)
	if err != nil {
		panic(fmt.Sprintf("pattern %q reached matching without being validated: %v", pattern, err))
	}
	return covered
}
