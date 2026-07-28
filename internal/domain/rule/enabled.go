package rule

// Names picks the rules to run: those asked for by name, else those the config
// enables, else every rule in the directory.
func Names(rulesDir string, explicit, enabled []string) ([]string, error) {
	if len(explicit) > 0 {
		return explicit, nil
	}
	if len(enabled) > 0 {
		return enabled, nil
	}
	return List(rulesDir)
}

// LoadAll loads each named rule, failing on the first that cannot be read.
func LoadAll(rulesDir string, names, knownTargets []string) ([]Rule, error) {
	rules := make([]Rule, 0, len(names))
	for _, name := range names {
		loaded, err := Load(rulesDir, name, knownTargets)
		if err != nil {
			return nil, err
		}
		rules = append(rules, loaded)
	}
	return rules, nil
}
