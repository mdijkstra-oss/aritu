package report

import "strings"

func RenderCompact(lines []string) string {
	return strings.Join(lines, "; ")
}

func RenderFull(lines []string) string {
	return strings.Join(lines, "\n\n") + "\n"
}
