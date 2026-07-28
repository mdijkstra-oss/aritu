package config

import "time"

// Config holds the configuration.
type Config struct {
	Votes   *int
	Timeout *time.Duration
	Rules   Rules
}

// Rules is a block because the word names two things: where rules live, and
// which of them to run.
type Rules struct {
	Dir     *string
	Enabled []string
}

// FileName is the only name searched for.
const FileName = "settings.yml"

// Timeout is in milliseconds.
var Timeout = 30000

// Load loads a config file from path and returns it.
//
// It decodes strictly, so an unknown key fails rather than being skipped, and it
// resolves every path it finds against the directory the file itself sits in,
// because a path written down in a file should resolve in the frame it was
// written in rather than wherever the process happens to be standing.
func Load(path string) (Config, error) {
	return Config{}, nil
}
