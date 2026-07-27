package scenario

import "strings"

type Index struct {
	postings map[string][]int
}

func New() *Index {
	return &Index{postings: map[string][]int{}}
}

func (i *Index) Add(docID int, text string) {
	for _, word := range strings.Split(text, " ") {
		term := stem(word)
		if term == "" {
			continue
		}
		i.postings[term] = append(i.postings[term], docID)
	}
}

func (i *Index) Search(query string) []int {
	return i.postings[stem(query)]
}

func stem(word string) string {
	return strings.TrimSuffix(strings.ToLower(word), "s")
}
