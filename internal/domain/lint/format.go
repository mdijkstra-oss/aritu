package lint

import (
	"fmt"
	"io"
	"maps"
	"slices"
	"strings"
)

func Format(w io.Writer, r Report, colour bool) error {
	pen := pen{colours: paletteFor(colour)}
	p := pen.colours

	fmt.Fprintf(&pen.buf, "%s%s%s  %s%s%s\n\n", p.bold, r.Rule, p.reset, p.dim, r.File, p.reset)

	if r.Error != "" {
		fmt.Fprintf(&pen.buf, "  %s✗ could not run%s\n    %s%s%s\n\n", p.fail, p.reset, p.dim, r.Error, p.reset)
		_, err := io.WriteString(w, pen.buf.String())
		return err
	}
	if len(r.Verdicts) == 0 {
		fmt.Fprintf(&pen.buf, "  %sno units to judge%s\n\n", p.dim, p.reset)
		_, err := io.WriteString(w, pen.buf.String())
		return err
	}

	for _, group := range groupsOf(r.Verdicts) {
		writeGroup(&pen, group, r)
	}
	writeSummary(&pen, r)

	_, err := io.WriteString(w, pen.buf.String())
	return err
}

type pen struct {
	buf     strings.Builder
	colours palette
}

type Outcome int

const (
	OutcomePass Outcome = iota + 1
	OutcomeSplit
	OutcomeFail
)

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

type reportCase struct {
	Unit  string
	Label string
	Count int
}

func splitUnit(name string) (function, caseName string, hasCase bool) {
	open := strings.LastIndex(name, " (")
	if open < 0 || !strings.HasSuffix(name, ")") {
		return name, "", false
	}
	return name[:open], name[open+2 : len(name)-1], true
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

func writeGroup(w *pen, group reportGroup, r Report) {
	if len(group.Cases) == 0 {
		if OutcomeFor(group.Count, r.Votes) == OutcomePass {
			return
		}
		writeUnit(w, "  ", reportCase{Unit: group.Unit, Label: group.Function, Count: group.Count}, r)
		w.buf.WriteString("\n")
		return
	}
	fallen := filterFallen(group.Cases, r.Votes)
	if len(fallen) == 0 {
		return
	}
	fmt.Fprintf(&w.buf, "  %s%s%s\n", w.colours.bold, group.Function, w.colours.reset)
	for _, c := range fallen {
		writeUnit(w, "    ", c, r)
	}
	w.buf.WriteString("\n")
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

func writeUnit(w *pen, indent string, c reportCase, r Report) {
	p := w.colours
	outcome := OutcomeFor(c.Count, r.Votes)
	fmt.Fprintf(&w.buf, "%s%s%s%s %s", indent, colourOf(p, outcome), markOf(outcome), p.reset, c.Label)
	if c.Count > 0 && c.Count < r.Votes {
		fmt.Fprintf(&w.buf, " %s(%d of %d)%s", p.dim, c.Count, r.Votes, p.reset)
	}
	w.buf.WriteString("\n")
	for _, reason := range reasonsFor(r, c.Unit, c.Count) {
		fmt.Fprintf(&w.buf, "%s  %s%s%s\n", indent, p.reason, reason, p.reset)
	}
}

func reasonsFor(r Report, unit string, count int) []string {
	if OutcomeFor(count, r.Votes) == OutcomePass {
		return nil
	}
	return r.Reasons[unit]
}

type tally struct {
	passed int
	split  int
	failed int
}

func tallyOf(r Report) tally {
	var counted tally
	for _, count := range r.Verdicts {
		switch OutcomeFor(count, r.Votes) {
		case OutcomePass:
			counted.passed++
		case OutcomeSplit:
			counted.split++
		case OutcomeFail:
			counted.failed++
		default:
			panic(fmt.Sprintf("unknown outcome for count %d", count))
		}
	}
	return counted
}

func writeSummary(w *pen, r Report) {
	p := w.colours
	counted := tallyOf(r)

	parts := []string{fmt.Sprintf("%s%d passed%s", p.pass, counted.passed, p.reset)}
	if fell := counted.failed + counted.split; fell > 0 {
		parts = append(parts, fmt.Sprintf("%s%d failed%s", p.fail, fell, p.reset))
	}
	if counted.split > 0 {
		parts = append(parts, fmt.Sprintf("%s%d split%s", p.split, counted.split, p.reset))
	}
	parts = append(parts, fmt.Sprintf("%s%s, %s%s", p.dim, plural(len(r.Verdicts), "unit"), plural(r.Votes, "vote"), p.reset))
	fmt.Fprintf(&w.buf, "  %s\n", strings.Join(parts, p.dim+"  ·  "+p.reset))
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
