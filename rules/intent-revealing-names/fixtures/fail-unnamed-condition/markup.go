package markup

import "strings"

func Strip(fragment string) string {
	if strings.HasPrefix(fragment, "<") && strings.HasSuffix(fragment, ">") && len(fragment) > 2 && !strings.HasPrefix(fragment, "</") {
		return fragment[1 : len(fragment)-1]
	}
	return fragment
}
