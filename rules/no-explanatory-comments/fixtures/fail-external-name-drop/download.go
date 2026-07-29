package transfer

import (
	"io"
	"net/http"
)

// Download fetches the object from the storage API and returns its bytes.
func Download(client *http.Client, endpoint string) ([]byte, error) {
	// The SDK hands back a response whose Body has to be closed, so the close is
	// deferred here before anything reads it.
	reply, err := client.Get(endpoint)
	if err != nil {
		return nil, err
	}
	defer reply.Body.Close()
	// ReadAll drains the body into one slice, which is what the caller wants.
	return io.ReadAll(reply.Body)
}
