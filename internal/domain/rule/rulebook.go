package rule

import "strings"

// Rulebook renders the enabled rules as the document an agent is handed before it
// writes anything: one heading per rule, and that rule's description under it.
//
// The criterion in prompt.md is deliberately left out. It is written to settle a
// verdict about a file that already exists — two poles, near-misses, what not to
// reject — and it is long, adversarial and organised around the ways a judge goes
// wrong. Handing that to whoever is about to write the file is handing them the
// argument instead of the instruction. The description is the same property stated
// as the work it takes to comply.
//
// The order is the order the rules were given, which is the order they were
// enabled, so the same rule set always renders the same document.
func Rulebook(rules []Rule) string {
	sections := make([]string, 0, len(rules)+1)
	sections = append(sections, rulebookPreamble)
	for _, judged := range rules {
		sections = append(sections, sectionFor(judged))
	}
	return strings.Join(sections, "\n\n") + "\n"
}

const rulebookPreamble = `# Coding rules

The following rules must be abided by. Read them before writing anything they are
about, and check what you wrote against them before calling it done.`

func sectionFor(judged Rule) string {
	return "## " + judged.Name + "\n\n" + judged.Description
}
