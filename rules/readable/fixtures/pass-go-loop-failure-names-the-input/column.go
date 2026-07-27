package scenario

func Label(index int) string {
	label := ""
	for remaining := index + 1; remaining > 0; remaining = (remaining - 1) / alphabetSize {
		label = string(rune('A'+(remaining-1)%alphabetSize)) + label
	}
	return label
}

const alphabetSize = 26
