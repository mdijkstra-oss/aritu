package rule

type Selection struct {
	RulesDir string
	Explicit []string
	Enabled  []string
}

// Names picks the rules to run: those asked for by name, else those the config
// enables, else every rule in the directory.
func Names(selection Selection) ([]string, error) {
	if len(selection.Explicit) > 0 {
		return selection.Explicit, nil
	}
	if len(selection.Enabled) > 0 {
		return selection.Enabled, nil
	}
	return List(selection.RulesDir)
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
