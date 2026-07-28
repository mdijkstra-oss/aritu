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
// The bands run hardest first, and within one the order is the order the rules
// were given, so the same rule set always renders the same document. Banding is
// what a flat list cannot say: which of two rules to satisfy first when a file
// breaks both.
func Rulebook(rules []Rule) string {
	sections := make([]string, 0, len(rules)+len(priorityBands)+1)
	sections = append(sections, rulebookPreamble)
	for _, band := range priorityBands {
		banded := rulesAt(rules, band.Priority)
		if len(banded) == 0 {
			continue
		}
		sections = append(sections, "## "+band.Title+"\n\n"+band.Gloss)
		for _, judged := range banded {
			sections = append(sections, SectionAt(judged, bandedDepth))
		}
	}
	return strings.Join(sections, "\n\n") + "\n"
}

// Section is one rule as a titled block of markdown. Both readers get it from
// here: the rulebook stacks these into a document, and the verdict prompt frames
// one of them for the model. What is asked of a writer and what is asked of the
// judge are then the same text under the same heading, down to the wording.
func Section(judged Rule) string {
	return SectionAt(judged, sectionDepth)
}

// SectionAt exists because the rulebook's rules hang under a band heading and
// the verdict prompt's do not. Only the depth moves: one renderer means the
// criterion cannot differ between the two readers.
func SectionAt(judged Rule, depth int) string {
	return strings.Repeat("#", depth) + " " + Title(judged.Name) + "\n\n" + strings.TrimSpace(judged.Prompt)
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

const (
	sectionDepth = 2
	bandedDepth  = 3
)

type priorityBand struct {
	Priority Priority
	Title    string
	Gloss    string
}

var priorityBands = []priorityBand{
	{
		Priority: PrioritySevere,
		Title:    "Severe",
		Gloss:    "Fix these before anything below them. The fix relocates the code around it, and findings nested inside one often go with it.",
	},
	{
		Priority: PriorityHigh,
		Title:    "High",
		Gloss:    "A shape callers depend on. The fix reaches past the declaration that carries it, so it lands after the structural work above.",
	},
	{
		Priority: PriorityMed,
		Title:    "Medium",
		Gloss:    "Local enough to fix where it stands: a rename, a move, a deletion. Nothing here blocks the work above it.",
	},
}

func rulesAt(rules []Rule, priority Priority) []Rule {
	banded := make([]Rule, 0, len(rules))
	for _, judged := range rules {
		if judged.Priority.Band() == priority {
			banded = append(banded, judged)
		}
	}
	return banded
}

const rulebookPreamble = `# Coding rules

The following rules must be abided by. Read them before writing anything they are
about, and check what you wrote against them before calling it done.

They are grouped by what a violation costs. Where a file breaks rules in more
than one band, the higher band goes first: its fix moves the code the lower ones
are about.`
