package merge

// Defaults copies each key the target is missing. Mutating the target in
// place is the right call here: every caller builds the map on the line
// above the call, so nothing else can be holding a reference, and returning
// a copy would cost an allocation for no reader's benefit.
func Defaults(target, defaults map[string]string) {
	for key, value := range defaults {
		if _, ok := target[key]; !ok {
			target[key] = value
		}
	}
}
