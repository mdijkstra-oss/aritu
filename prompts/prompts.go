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

// render substitutes every {{name}} in a template. What is checked for a
// placeholder nobody supplied is the template, never the result: the values carry
// source code, and a file holding braces of its own — a Go composite literal such
// as []Param{{...}}, a Handlebars view, a Go template — is the ordinary input this
// tool exists to read rather than a prompt this package failed to fill.
func render(name string, values map[string]string) string {
	template := read(name)
	if unfilled, hasUnfilled := findUnfilled(template, values); hasUnfilled {
		panic(fmt.Sprintf("prompt %s leaves %s unfilled", name, unfilled))
	}
	rendered := template
	for key, value := range values {
		rendered = strings.ReplaceAll(rendered, "{{"+key+"}}", value)
	}
	return rendered
}

// findUnfilled reports the first placeholder the template carries that no value
// answers, so a typo in a template is still caught before the braces reach a model.
// An opening brace the template never closes counts as one.
func findUnfilled(template string, values map[string]string) (string, bool) {
	for rest := template; ; {
		opened := strings.Index(rest, "{{")
		if opened < 0 {
			return "", false
		}
		rest = rest[opened+2:]
		closed := strings.Index(rest, "}}")
		if closed < 0 {
			return "{{", true
		}
		if key := rest[:closed]; !isSupplied(key, values) {
			return "{{" + key + "}}", true
		}
		rest = rest[closed+2:]
	}
}

func isSupplied(key string, values map[string]string) bool {
	_, supplied := values[key]
	return supplied
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
