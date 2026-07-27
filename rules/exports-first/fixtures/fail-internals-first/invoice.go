package invoice

import "strings"

func normaliseNumber(raw string) string {
	return strings.ToUpper(strings.TrimSpace(raw))
}

const numberPrefix = "INV-"

type Invoice struct {
	Number string
	Total  int
}

func NewInvoice(raw string, total int) Invoice {
	return Invoice{Number: numberPrefix + normaliseNumber(raw), Total: total}
}
