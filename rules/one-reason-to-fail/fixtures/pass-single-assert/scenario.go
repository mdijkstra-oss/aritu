package scenario

func Lookup(settings map[string]string, key, fallback string) string {
	if value, isPresent := settings[key]; isPresent {
		return value
	}
	return fallback
}
