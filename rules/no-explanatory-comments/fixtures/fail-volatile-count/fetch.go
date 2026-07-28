package fetch

// attempts is how many turns one request gets. With the current three
// callers, none of them batching more than a dozen rows, even worst-case
// retries add under a second of latency.
const attempts = 3

func withRetry(call func() error) error {
	var err error
	for turn := 0; turn < attempts; turn++ {
		if err = call(); err == nil {
			return nil
		}
	}
	return err
}
