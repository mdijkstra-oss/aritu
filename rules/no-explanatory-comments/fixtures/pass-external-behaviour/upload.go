package transfer

import (
	"errors"
	"io"
	"net/http"
)

var errRejected = errors.New("upload rejected")

// The API requires every multipart part to carry a name.
const partName = "file"

// The storage client already paces 408, 429 and 5xx with backoff of its own.
func Retrying(client *http.Client, attempts int) func(string, io.Reader) error {
	return func(endpoint string, body io.Reader) error {
		var err error
		for range attempts {
			err = upload(client, endpoint, body)
			if !errors.Is(err, errRejected) {
				return err
			}
		}
		return err
	}
}

func upload(client *http.Client, endpoint string, body io.Reader) error {
	request, err := http.NewRequest(http.MethodPost, endpoint+"?part="+partName, body)
	if err != nil {
		return err
	}
	reply, err := client.Do(request)
	if err != nil {
		return err
	}
	defer reply.Body.Close()
	if reply.StatusCode >= 400 {
		return errRejected
	}
	return nil
}
