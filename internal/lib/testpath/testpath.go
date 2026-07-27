// Package testpath answers, from a path alone, whether a file is a test and where
// the implementation it covers could sit. It holds one table of naming conventions
// and touches no filesystem: which of the candidates it offers actually exists is
// the caller's question to ask.
package testpath

import (
	"fmt"
	"maps"
	"path/filepath"
	"slices"
	"strings"
)

// IsTestFile reports whether a path names a test file under any convention in the
// table. A file is a test either because its own name carries a test affix or
// because it sits in a directory that marks everything inside it as tests.
func IsTestFile(path string) bool {
	found, isKnown := conventionFor(path)
	return isKnown && len(sourceStemsFor(found, path)) > 0
}

// Extensions lists every file extension the conventions table covers, sorted. It
// reads the same index the table is looked up through, so an ecosystem added as a
// row widens it with no second edit.
func Extensions() []string {
	return slices.Sorted(maps.Keys(conventionsByExtension))
}

// SourceCandidates lists where the implementation a test file covers could sit,
// most likely first, and nothing at all when the path is not a test file.
//
// Resolution is a search rather than a derivation because a source tree mirrored
// beside the test tree cannot be reached by swapping a suffix: the file beside the
// test and the file under the parallel root are both plausible, and only the
// filesystem settles which one is there.
func SourceCandidates(testPath string) []string {
	found, isKnown := conventionFor(testPath)
	if !isKnown {
		return nil
	}
	stems := sourceStemsFor(found, testPath)
	if len(stems) == 0 {
		return nil
	}

	extension := filepath.Ext(testPath)
	dirs := searchDirsFor(found, filepath.Dir(testPath))
	candidates := make([]string, 0, len(stems)*len(dirs))
	claimed := map[string]bool{filepath.Clean(testPath): true}
	for _, dir := range dirs {
		for _, stem := range stems {
			candidate := filepath.Join(dir, stem+extension)
			if claimed[candidate] {
				continue
			}
			claimed[candidate] = true
			candidates = append(candidates, candidate)
		}
	}
	return candidates
}

// convention is one ecosystem's test-file naming, held as data so that supporting
// a layout is a row in the table rather than a branch in the code.
type convention struct {
	extensions []string
	affixes    []affix
	testDirs   []string
	moves      []move
}

// affix is a fragment of a file's stem that marks the file as a test.
type affix struct {
	text string
	at   placement
}

// move rewrites a run of directory segments, which is what reaches a mirrored
// source tree. An empty to drops the segments rather than replacing them.
type move struct {
	from []string
	to   []string
}

type placement int

const (
	prefix placement = iota + 1
	suffix
)

// conventions is the whole of what aritu knows about test-file layouts. The order
// of affixes and moves within a row is the order candidates come back in, so the
// most common layout for each ecosystem is listed first.
var conventions = []convention{
	{
		extensions: []string{".go"},
		affixes:    []affix{{text: "_test", at: suffix}},
	},
	{
		extensions: []string{".ts", ".tsx", ".mts", ".cts", ".js", ".jsx", ".mjs", ".cjs"},
		affixes:    []affix{{text: ".test", at: suffix}, {text: ".spec", at: suffix}},
		testDirs:   []string{"__tests__", "__test__"},
		moves: []move{
			{from: []string{"__tests__"}},
			{from: []string{"__test__"}},
			{from: []string{"test"}, to: []string{"src"}},
			{from: []string{"tests"}, to: []string{"src"}},
			{from: []string{"spec"}, to: []string{"src"}},
		},
	},
	{
		extensions: []string{".py"},
		affixes:    []affix{{text: "test_", at: prefix}, {text: "_test", at: suffix}},
		moves: []move{
			{from: []string{"tests"}},
			{from: []string{"test"}},
			{from: []string{"tests"}, to: []string{"src"}},
			{from: []string{"test"}, to: []string{"src"}},
		},
	},
	{
		extensions: []string{".java"},
		affixes:    []affix{{text: "Tests", at: suffix}, {text: "Test", at: suffix}, {text: "Test", at: prefix}},
		moves:      []move{{from: []string{"src", "test", "java"}, to: []string{"src", "main", "java"}}},
	},
}

var conventionsByExtension = indexByExtension(conventions)

var affixCutters = map[placement]func(string, string) (string, bool){
	prefix: strings.CutPrefix,
	suffix: strings.CutSuffix,
}

func indexByExtension(rows []convention) map[string]convention {
	indexed := make(map[string]convention, len(rows))
	for _, row := range rows {
		for _, extension := range row.extensions {
			indexed[extension] = row
		}
	}
	return indexed
}

func conventionFor(path string) (convention, bool) {
	found, isKnown := conventionsByExtension[filepath.Ext(path)]
	return found, isKnown
}

// sourceStemsFor derives the base names the implementation could carry. A file in a
// directory that marks its contents as tests contributes its own stem too, because
// there the directory carries the marking the file name does not.
func sourceStemsFor(c convention, path string) []string {
	base := filepath.Base(path)
	stem := strings.TrimSuffix(base, filepath.Ext(base))

	stems := make([]string, 0, len(c.affixes)+1)
	for _, marker := range c.affixes {
		trimmed, isMarked := cutAffix(stem, marker)
		if isMarked && trimmed != "" {
			stems = appendUnseen(stems, trimmed)
		}
	}
	if isInTestDir(c, path) && stem != "" {
		stems = appendUnseen(stems, stem)
	}
	return stems
}

func cutAffix(stem string, marker affix) (string, bool) {
	cut, isKnown := affixCutters[marker.at]
	if !isKnown {
		panic(fmt.Sprintf("unknown affix placement: %d", int(marker.at)))
	}
	return cut(stem, marker.text)
}

func isInTestDir(c convention, path string) bool {
	return slices.Contains(c.testDirs, filepath.Base(filepath.Dir(path)))
}

// searchDirsFor lists the directories to look in, the test's own first. A sibling
// implementation is both the commonest layout and the cheapest to confirm.
func searchDirsFor(c convention, dir string) []string {
	dirs := []string{dir}
	segments := segmentsOf(dir)
	for _, rewrite := range c.moves {
		moved, isMoved := applyMove(segments, rewrite)
		if !isMoved {
			continue
		}
		dirs = appendUnseen(dirs, moved)
	}
	return dirs
}

func segmentsOf(dir string) []string {
	return strings.Split(filepath.ToSlash(dir), "/")
}

func applyMove(segments []string, rewrite move) (string, bool) {
	at := lastIndexOfRun(segments, rewrite.from)
	if at < 0 {
		return "", false
	}
	moved := make([]string, 0, len(segments)-len(rewrite.from)+len(rewrite.to))
	moved = append(moved, segments[:at]...)
	moved = append(moved, rewrite.to...)
	moved = append(moved, segments[at+len(rewrite.from):]...)
	return filepath.FromSlash(strings.Join(moved, "/")), true
}

// lastIndexOfRun finds the run nearest the file, so a repository that nests a
// tests directory inside another one rewrites the inner occurrence.
func lastIndexOfRun(segments, run []string) int {
	for at := len(segments) - len(run); at >= 0; at-- {
		if slices.Equal(segments[at:at+len(run)], run) {
			return at
		}
	}
	return -1
}

func appendUnseen(items []string, item string) []string {
	if slices.Contains(items, item) {
		return items
	}
	return append(items, item)
}
