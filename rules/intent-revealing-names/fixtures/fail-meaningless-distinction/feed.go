package feed

func Merge(data1 []string, data2 []string) []string {
	merged := make([]string, 0, len(data1)+len(data2))
	merged = append(merged, data1...)
	merged = append(merged, data2...)
	return merged
}
