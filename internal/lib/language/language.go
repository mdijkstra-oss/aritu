// Package language answers which files hold source code, by extension, for a
// linter that judges prose about code rather than the code's own grammar.
//
// The table is grouped by language so that adding one is a single edit, and it
// is deliberately a list rather than a detector. Nothing here parses a file or
// reads a shebang: a repository whose language is missing gets a hard "no
// enabled rule targets" from the sweep, which is a better failure than a file
// silently skipped.
package language

import (
	"path/filepath"
	"slices"
)

func IsSourceFile(path string) bool {
	return slices.Contains(extensions, filepath.Ext(path))
}

func Extensions() []string {
	return slices.Clone(extensions)
}

type language struct {
	Name       string
	Extensions []string
}

var languages = []language{
	{"Go", []string{".go"}},
	{"Python", []string{".py", ".pyi"}},
	{"JavaScript", []string{".js", ".jsx", ".mjs", ".cjs"}},
	{"TypeScript", []string{".ts", ".tsx", ".mts", ".cts"}},
	{"Vue", []string{".vue"}},
	{"Svelte", []string{".svelte"}},
	{"Java", []string{".java"}},
	{"Kotlin", []string{".kt", ".kts"}},
	{"Scala", []string{".scala", ".sc"}},
	{"Groovy", []string{".groovy"}},
	{"C", []string{".c", ".h"}},
	{"C++", []string{".cc", ".cpp", ".cxx", ".hh", ".hpp", ".hxx"}},
	{"C#", []string{".cs"}},

	// .m is Objective-C here and MATLAB elsewhere, and telling them apart needs
	// the file's contents. Objective-C wins because the alternative is skipping
	// every iOS repository silently.
	{"Objective-C", []string{".m", ".mm"}},

	{"Swift", []string{".swift"}},
	{"Rust", []string{".rs"}},
	{"Zig", []string{".zig"}},
	{"Ruby", []string{".rb"}},
	{"PHP", []string{".php"}},

	// .pl is Perl here and Prolog elsewhere, resolved the same way and for the
	// same reason as .m above.
	{"Perl", []string{".pl", ".pm"}},

	{"Lua", []string{".lua"}},
	{"Dart", []string{".dart"}},
	{"Elixir", []string{".ex", ".exs"}},
	{"Erlang", []string{".erl", ".hrl"}},
	{"Haskell", []string{".hs"}},
	{"Clojure", []string{".clj", ".cljs", ".cljc"}},
	{"OCaml", []string{".ml", ".mli"}},
	{"F#", []string{".fs", ".fsi", ".fsx"}},
	{"Julia", []string{".jl"}},

	// Both cases are listed because extensions are matched exactly, and R's own
	// convention is the capital.
	{"R", []string{".r", ".R"}},

	{"Shell", []string{".sh", ".bash", ".zsh"}},
	{"PowerShell", []string{".ps1", ".psm1"}},
	{"Solidity", []string{".sol"}},
}

var extensions = extensionsOf(languages)

func extensionsOf(rows []language) []string {
	flattened := make([]string, 0, len(rows))
	for _, row := range rows {
		flattened = append(flattened, row.Extensions...)
	}
	return flattened
}
