package lines

const enableCSV = false

func Render(rows []string, delimiter string) string {
	if enableCSV {
		return renderCSV(rows)
	}
	rendered := ""
	for _, row := range rows {
		rendered += row + "\n"
	}
	return rendered
}

func renderCSV(rows []string) string {
	rendered := ""
	for _, row := range rows {
		rendered += row + ",\n"
	}
	return rendered
}
