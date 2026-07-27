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
	"io/fs"
	"path"
	"slices"
	"strings"
)

// Verdict renders the prompt that asks for one verdict per unit: the generic
// framing, then whatever fragments the rule asked to include, then the rule.
func Verdict(includes []string, rulePrompt, units, sources string) string {
	return render("verdict.md", map[string]string{
		"fragments": join(includes, verdictFragment),
		"rule":      strings.TrimSpace(rulePrompt),
		"units":     strings.TrimSpace(units),
		"sources":   strings.TrimSpace(sources),
	})
}

// Enumerate renders the prompt that lists a file's units. It carries no rule, only
// the same fragments: the units of a file are the same whichever rule is about to
// judge them, which is what lets one enumeration serve every rule that includes the
// same fragments.
func Enumerate(includes []string, source string) string {
	return render("enumerate.md", map[string]string{
		"fragments": join(includes, enumerateFragment),
		"source":    strings.TrimSpace(source),
	})
}

// Known names every fragment a rule may include, sorted. Rules are loaded from a
// repository and prompts are not, so an include naming a fragment this binary does
// not carry has to be caught when the rule is read.
func Known() []string {
	entries, err := fs.ReadDir(files, fragmentDir)
	if err != nil {
		panic(fmt.Sprintf("embedded fragment directory: %v", err))
	}
	names := make([]string, 0, len(entries))
	for _, entry := range entries {
		name, isVerdict := strings.CutSuffix(entry.Name(), ".md")
		if !isVerdict || strings.HasSuffix(name, enumerateSuffix) {
			continue
		}
		names = append(names, name)
	}
	slices.Sort(names)
	return names
}

// IsKnown reports whether a rule may include the named fragment.
func IsKnown(name string) bool {
	return slices.Contains(Known(), name)
}

//go:embed *.md fragments/*.md
var files embed.FS

const (
	fragmentDir     = "fragments"
	enumerateSuffix = ".enumerate"
)

// verdictFragment is what a fragment contributes to a verdict call, and
// enumerateFragment what it contributes to an enumeration.
//
// The enumeration gets both halves: what a test, a scope and a case are is the same
// answer whether the model is listing them or judging them, and only the listing
// half — what to write down and what to leave out — is enumeration's alone. A
// fragment with nothing to say about listing has no second file.
func verdictFragment(name string) string {
	return read(path.Join(fragmentDir, name+".md"))
}

func enumerateFragment(name string) string {
	shared := verdictFragment(name)
	listing := readOptional(path.Join(fragmentDir, name+enumerateSuffix+".md"))
	if listing == "" {
		return shared
	}
	return shared + "\n\n" + listing
}

func join(includes []string, contribution func(string) string) string {
	parts := make([]string, 0, len(includes))
	for _, name := range includes {
		if part := strings.TrimSpace(contribution(name)); part != "" {
			parts = append(parts, part)
		}
	}
	return strings.Join(parts, "\n\n")
}

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
	return collapseBlankRuns(rendered)
}

// collapseBlankRuns closes the gap a fragment nobody included leaves behind, so an
// omitted include costs no stray blank lines in the prompt.
func collapseBlankRuns(text string) string {
	for strings.Contains(text, "\n\n\n") {
		text = strings.ReplaceAll(text, "\n\n\n", "\n\n")
	}
	return strings.TrimSpace(text) + "\n"
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

func readOptional(name string) string {
	raw, err := files.ReadFile(name)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(raw))
}
