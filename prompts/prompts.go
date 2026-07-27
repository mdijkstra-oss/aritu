// Package prompts holds every prompt aritu sends, as markdown beside the code
// rather than as string literals inside it.
//
// They are embedded rather than read from disk. A rule set is something a
// repository configures and points at; a prompt is engine behaviour, and a binary
// whose question could change under it would report verdicts that mean something
// different from one machine to the next.
package prompts

import (
	"embed"
	"fmt"
	"strings"
)

// Verdict renders the prompt that asks for one verdict per unit. The layering runs
// generic to specific: what judging is at all, then what the units of this kind of
// file are, then the one rule being judged.
func Verdict(rulePrompt, units, sources string) string {
	return render("verdict.md", map[string]string{
		"unit_model": read(unitModelFile),
		"rule":       strings.TrimSpace(rulePrompt),
		"units":      strings.TrimSpace(units),
		"sources":    strings.TrimSpace(sources),
	})
}

// Enumerate renders the prompt that lists a file's units. It carries no rule: the
// units of a file are the same whichever rule is about to judge them, which is what
// lets one enumeration serve every rule in a run.
func Enumerate(source string) string {
	return render("enumerate.md", map[string]string{
		"unit_model":       read(unitModelFile),
		"unit_enumeration": read(unitEnumerationFile),
		"source":           strings.TrimSpace(source),
	})
}

//go:embed *.md units/*.md
var files embed.FS

// unitModelFile and unitEnumerationFile name the one unit model aritu ships. The
// seam is here rather than in configuration: a second model would be a second pair
// of files and a way to choose between them, and until one exists a setting with a
// single legal value is surface for nothing.
const (
	unitModelFile       = "units/tests.md"
	unitEnumerationFile = "units/tests-enumerate.md"
)

// render substitutes every {{name}} in a template. A placeholder the caller did not
// supply, or a value carrying a placeholder of its own, would both put braces in
// front of the model, so neither is tolerated.
func render(name string, values map[string]string) string {
	rendered := read(name)
	for key, value := range values {
		rendered = strings.ReplaceAll(rendered, "{{"+key+"}}", value)
	}
	if at := strings.Index(rendered, "{{"); at >= 0 {
		panic(fmt.Sprintf("prompt %s left the placeholder at offset %d unfilled", name, at))
	}
	return rendered
}

// read panics because the templates are embedded: a name that does not resolve is a
// typo in this package, caught by the first test that renders anything.
func read(name string) string {
	raw, err := files.ReadFile(name)
	if err != nil {
		panic(fmt.Sprintf("embedded prompt %s: %v", name, err))
	}
	return strings.TrimSpace(string(raw))
}
