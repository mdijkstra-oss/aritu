package invoice

import "strings"

type Invoice struct {
	Number string
	Total  int
}

const NumberPrefix = "INV-"

func NewInvoice(raw string, total int) Invoice {
	return Invoice{Number: NumberPrefix + normaliseNumber(raw), Total: total}
}

func normaliseNumber(raw string) string {
	return strings.ToUpper(strings.TrimSpace(raw))
}
