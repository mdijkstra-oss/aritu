package rule

import (
	"strings"
	"unicode"
)

// Rulebook renders the enabled rules as the document handed to whoever is about to
// write something, before anything exists to judge.
//
// What it writes is the rules themselves, in full. There is no second, friendlier
// statement of a rule kept beside the real one: a summary and a criterion drift
// apart the moment either is edited, and the drift is invisible because nothing
// compares them. One text, read by the model that judges and by the person that
// writes, cannot disagree with itself.
//
// The order is the order the rules were given, which is the order they were
// enabled, so the same rule set always renders the same document.
func Rulebook(rules []Rule) string {
	sections := make([]string, 0, len(rules)+1)
	sections = append(sections, rulebookPreamble)
	for _, judged := range rules {
		sections = append(sections, Section(judged))
	}
	return strings.Join(sections, "\n\n") + "\n"
}

// Section is one rule as a titled block of markdown. Both readers get it from
// here: the rulebook stacks these into a document, and the verdict prompt frames
// one of them for the model. What is asked of a writer and what is asked of the
// judge are then the same text under the same heading, down to the wording.
func Section(judged Rule) string {
	return "## " + Title(judged.Name) + "\n\n" + strings.TrimSpace(judged.Prompt)
}

// Title reads a rule's heading off its directory name, so a rule has one name and
// not two: my-pretty-rule is "My pretty rule". Any parking prefix comes off first —
// that is bookkeeping about whether the rule is enforced, and says nothing about
// what the rule asks for.
func Title(name string) string {
	spaced := strings.NewReplacer("-", " ", "_", " ").Replace(strings.TrimLeft(name, parkedPrefix))
	runes := []rune(strings.TrimSpace(spaced))
	if len(runes) == 0 {
		return ""
	}
	runes[0] = unicode.ToUpper(runes[0])
	return string(runes)
}

const rulebookPreamble = `# Coding rules

The following rules must be abided by. Read them before writing anything they are
about, and check what you wrote against them before calling it done.`
