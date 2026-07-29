package testpath

import (
	"fmt"
	"maps"
	"path/filepath"
	"slices"
	"strings"
)

func IsTestFile(path string) bool {
	found, isKnown := conventionFor(path)
	return isKnown && len(sourceStemsFor(found, path)) > 0
}

func Extensions() []string {
	return slices.Sorted(maps.Keys(conventionsByExtension))
}

func SourceCandidates(testPath string) []string {
	found, isKnown := conventionFor(testPath)
	if !isKnown {
		return nil
	}
	stems := sourceStemsFor(found, testPath)
	if len(stems) == 0 {
		return nil
	}

	dirs := searchDirsFor(found, filepath.Dir(testPath))
	return pathsAcross(dirs, stems, filepath.Ext(testPath), filepath.Clean(testPath))
}

func pathsAcross(dirs, stems []string, extension, excluded string) []string {
	paths := make([]string, 0, len(stems)*len(dirs))
	claimed := map[string]bool{excluded: true}
	for _, dir := range dirs {
		for _, stem := range stems {
			path := filepath.Join(dir, stem+extension)
			if claimed[path] {
				continue
			}
			claimed[path] = true
			paths = append(paths, path)
		}
	}
	return paths
}

type convention struct {
	extensions []string
	affixes    []affix
	testDirs   []string
	moves      []move
}

type affix struct {
	text string
	at   placement
}

type move struct {
	from []string
	to   []string
}

type placement int

const (
	prefix placement = iota + 1
	suffix
)

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
