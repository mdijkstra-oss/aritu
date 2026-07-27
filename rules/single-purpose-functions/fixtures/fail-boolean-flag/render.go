package report

import "strings"

func Render(lines []string, compact bool) string {
	if compact {
		return strings.Join(lines, "; ")
	}
	return strings.Join(lines, "\n\n") + "\n"
}
