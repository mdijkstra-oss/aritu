// Package prompts holds every prompt aritu sends, as markdown beside the code
// rather than as string literals inside it.
//
// They are embedded rather than read from disk. A rule set is something a
// repository configures and points at; a prompt is engine behaviour, and a binary
// whose question could change under it would report verdicts that mean something
// different from one machine to the next.
//
// The two model calls get a folder each: splitter/ lists a file's units, linter/
// judges them. Each folder holds the frame — instructions.md ahead of the work,
// task.md after it — and one file per unit kind, named for the granularity that
// selects it. A prompt is those pieces joined as named sections, so the frame can
// point at a section by name and the pieces never interpolate into each other.
package prompts

import (
	"embed"
	"fmt"
	"strings"
)

// File is one source file as a prompt shows it to the model.
type File struct {
	Path    string
	Content string
}

// Unit is one judged thing: Name is the identifier a reader sees when it fails,
// Key is what the model answers under. They differ because a key has to be a tidy
// JSON property while a name has to stay exactly what CI prints.
type Unit struct {
	Name string
	Key  string
}

// Linter renders the prompt that asks for one verdict per unit. The first file is
// the one under judgement; any file after it is context the rule asked to bring
// along, tagged apart so the model knows which one the units live in.
func Linter(kind, criterion string, units []Unit, files []File) string {
	sections := []string{
		section("instructions", read("linter/instructions.md")),
		section("unit", read("linter/"+kind+".md")),
		section("rule", criterion),
		section("units", unitLines(units)),
	}
	sections = append(sections, fileSections(files)...)
	sections = append(sections, section("task", read("linter/task.md")))
	return join(sections)
}

// Splitter renders the prompt that lists a file's units of one kind. It carries
// no rule: the units of a file are the same whichever rule is about to judge
// them, which is what lets one listing serve every rule at the same granularity.
func Splitter(kind string, file File) string {
	return join([]string{
		section("instructions", read("splitter/instructions.md")),
		section("unit", read("splitter/"+kind+".md")),
		fileSection("file", file),
		section("task", read("splitter/task.md")),
	})
}

//go:embed linter/*.md splitter/*.md
var files embed.FS

func section(name, body string) string {
	return "<" + name + ">\n" + strings.TrimSpace(body) + "\n</" + name + ">"
}

// fileSections tags the first file as the one under judgement and every later
// one as the source it was asked to bring along.
func fileSections(handed []File) []string {
	sections := make([]string, 0, len(handed))
	for at, f := range handed {
		tag := "file"
		if at > 0 {
			tag = "source"
		}
		sections = append(sections, fileSection(tag, f))
	}
	return sections
}

// fileSection carries the content verbatim: a file is the input this tool exists
// to read, never a template for it to fill.
func fileSection(tag string, f File) string {
	return fmt.Sprintf("<%s path=%q>\n%s\n</%s>", tag, f.Path, strings.Trim(f.Content, "\n"), tag)
}

func unitLines(units []Unit) string {
	lines := make([]string, 0, len(units))
	for _, unit := range units {
		lines = append(lines, fmt.Sprintf("%s   ->   %s", unit.Name, unit.Key))
	}
	return strings.Join(lines, "\n")
}

func join(sections []string) string {
	return strings.Join(sections, "\n\n") + "\n"
}

// read panics because the prompts are embedded: a name that does not resolve is a
// typo in this package or a granularity without its kind file, caught by the
// first test that renders anything.
func read(name string) string {
	raw, err := files.ReadFile(name)
	if err != nil {
		panic(fmt.Sprintf("embedded prompt %s: %v", name, err))
	}
	return strings.TrimSpace(string(raw))
}
