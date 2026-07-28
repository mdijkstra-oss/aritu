package cli

import "time"

type Flags struct {
	// Output is how to render the report: text or json.
	Output string `help:"How to render the report: text or json."`
	// Jobs is how many calls are allowed in flight at once.
	Jobs int `help:"Calls allowed in flight at once."`
	// Timeout is the deadline for the whole run.
	Timeout time.Duration `help:"Deadline for the whole run."`
}
