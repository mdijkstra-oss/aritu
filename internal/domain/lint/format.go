package lint

import (
	"fmt"
	"io"
	"maps"
	"slices"
	"strings"
)

// Format renders a report for a person rather than a parser: the units that fell
// short, grouped under the function that declares them, with the model's reason
// under each. Units that passed appear only in the closing count — a green row
// per unit says nothing the count does not, and on a corpus of any size those
// rows push the failures, the reason anyone is reading, off the screen.
//
// A count appears whenever the vote was not unanimous. The mark carries the
// decision; the count is how close it was, which is what a prompt being tuned
// needs to see.
func Format(w io.Writer, r Report, colour bool) error {
	p := paletteFor(colour)
	var b strings.Builder

	fmt.Fprintf(&b, "%s%s%s  %s%s%s\n\n", p.bold, r.Rule, p.reset, p.dim, r.File, p.reset)

	if r.Error != "" {
		fmt.Fprintf(&b, "  %s✗ could not run%s\n    %s%s%s\n\n", p.fail, p.reset, p.dim, r.Error, p.reset)
		_, err := io.WriteString(w, b.String())
		return err
	}
	if len(r.Verdicts) == 0 {
		fmt.Fprintf(&b, "  %sno units to judge%s\n\n", p.dim, p.reset)
		_, err := io.WriteString(w, b.String())
		return err
	}

	for _, group := range groupsOf(r.Verdicts) {
		writeGroup(&b, p, group, r, len(group.Cases) == 0)
	}
	writeSummary(&b, p, r)

	_, err := io.WriteString(w, b.String())
	return err
}

// Outcome is how one unit fared across the votes.
type Outcome int

const (
	// OutcomePass is a majority judging the unit to satisfy the rule.
	OutcomePass Outcome = iota + 1
	// OutcomeSplit is an exact tie, which fails: half the votes is not a majority.
	OutcomeSplit
	// OutcomeFail is a majority judging the unit to fall short.
	OutcomeFail
)

// OutcomeFor reads a unit's count against the votes it was given. A strict
// majority carries the unit; a tie fails it.
func OutcomeFor(count, votes int) Outcome {
	switch {
	case count*2 > votes:
		return OutcomePass
	case count*2 == votes:
		return OutcomeSplit
	default:
		return OutcomeFail
	}
}

type reportGroup struct {
	Function string
	Unit     string
	Count    int
	Cases    []reportCase
}

// reportCase keeps the full identifier alongside the label it prints, because
// reasons are keyed by identifier and two functions can carry the same case label.
type reportCase struct {
	Unit  string
	Label string
	Count int
}

func groupsOf(verdicts map[string]int) []reportGroup {
	byFunction := map[string]*reportGroup{}
	for _, unit := range slices.Sorted(maps.Keys(verdicts)) {
		function, caseName, hasCase := splitUnit(unit)
		group, seen := byFunction[function]
		if !seen {
			group = &reportGroup{Function: function}
			byFunction[function] = group
		}
		if !hasCase {
			group.Unit = unit
			group.Count = verdicts[unit]
			continue
		}
		group.Cases = append(group.Cases, reportCase{Unit: unit, Label: caseName, Count: verdicts[unit]})
	}

	groups := make([]reportGroup, 0, len(byFunction))
	for _, function := range slices.Sorted(maps.Keys(byFunction)) {
		groups = append(groups, *byFunction[function])
	}
	return groups
}

func writeGroup(b *strings.Builder, p palette, group reportGroup, r Report, standalone bool) {
	if standalone {
		if OutcomeFor(group.Count, r.Votes) == OutcomePass {
			return
		}
		writeUnit(b, p, "  ", group.Unit, group.Function, group.Count, r)
		b.WriteString("\n")
		return
	}
	fallen := filterFallen(group.Cases, r.Votes)
	if len(fallen) == 0 {
		return
	}
	fmt.Fprintf(b, "  %s%s%s\n", p.bold, group.Function, p.reset)
	for _, c := range fallen {
		writeUnit(b, p, "    ", c.Unit, c.Label, c.Count, r)
	}
	b.WriteString("\n")
}

func filterFallen(cases []reportCase, votes int) []reportCase {
	fallen := make([]reportCase, 0, len(cases))
	for _, c := range cases {
		if OutcomeFor(c.Count, votes) != OutcomePass {
			fallen = append(fallen, c)
		}
	}
	return fallen
}

func writeUnit(b *strings.Builder, p palette, indent, unit, label string, count int, r Report) {
	outcome := OutcomeFor(count, r.Votes)
	fmt.Fprintf(b, "%s%s%s%s %s", indent, colourOf(p, outcome), markOf(outcome), p.reset, label)
	if count > 0 && count < r.Votes {
		fmt.Fprintf(b, " %s(%d of %d)%s", p.dim, count, r.Votes, p.reset)
	}
	b.WriteString("\n")
	for _, reason := range reasonsFor(r, unit, count) {
		fmt.Fprintf(b, "%s  %s%s%s\n", indent, p.reason, reason, p.reset)
	}
}

// reasonsFor looks the explanations up by full identifier rather than by the
// printed label: two functions can each declare a case called "empty input", and
// matching on the label alone would attach one function's reason to the other's
// case, picked at random by map iteration order.
func reasonsFor(r Report, unit string, count int) []string {
	if OutcomeFor(count, r.Votes) == OutcomePass {
		return nil
	}
	return r.Reasons[unit]
}

func writeSummary(b *strings.Builder, p palette, r Report) {
	passed, split, failed := 0, 0, 0
	for _, count := range r.Verdicts {
		switch OutcomeFor(count, r.Votes) {
		case OutcomePass:
			passed++
		case OutcomeSplit:
			split++
		case OutcomeFail:
			failed++
		default:
			panic(fmt.Sprintf("unknown outcome for count %d", count))
		}
	}

	parts := []string{fmt.Sprintf("%s%d passed%s", p.pass, passed, p.reset)}
	if failed+split > 0 {
		parts = append(parts, fmt.Sprintf("%s%d failed%s", p.fail, failed+split, p.reset))
	}
	if split > 0 {
		parts = append(parts, fmt.Sprintf("%s%d split%s", p.split, split, p.reset))
	}
	parts = append(parts, fmt.Sprintf("%s%s, %s%s", p.dim, plural(len(r.Verdicts), "unit"), plural(r.Votes, "vote"), p.reset))
	fmt.Fprintf(b, "  %s\n", strings.Join(parts, p.dim+"  ·  "+p.reset))
}

func plural(n int, singular string) string {
	if n == 1 {
		return fmt.Sprintf("%d %s", n, singular)
	}
	return fmt.Sprintf("%d %ss", n, singular)
}

func markOf(outcome Outcome) string {
	switch outcome {
	case OutcomePass:
		return "✓"
	case OutcomeSplit:
		return "!"
	case OutcomeFail:
		return "✗"
	default:
		panic(fmt.Sprintf("unknown outcome: %d", int(outcome)))
	}
}

func colourOf(p palette, outcome Outcome) string {
	switch outcome {
	case OutcomePass:
		return p.pass
	case OutcomeSplit:
		return p.split
	case OutcomeFail:
		return p.fail
	default:
		panic(fmt.Sprintf("unknown outcome: %d", int(outcome)))
	}
}

// palette is empty for anything that is not a terminal, so a redirected run holds
// the same bytes a person saw, without escape sequences through the middle of them.
type palette struct {
	pass   string
	fail   string
	split  string
	reason string
	dim    string
	bold   string
	reset  string
}

func paletteFor(colour bool) palette {
	if !colour {
		return palette{}
	}
	return palette{
		pass:   "\x1b[32m",
		fail:   "\x1b[31m",
		split:  "\x1b[33m",
		reason: "\x1b[35m",
		dim:    "\x1b[2m",
		bold:   "\x1b[1m",
		reset:  "\x1b[0m",
	}
}
