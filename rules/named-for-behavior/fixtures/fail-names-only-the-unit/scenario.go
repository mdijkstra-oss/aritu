package scenario

import (
	"fmt"
	"strings"
)

func ParseAddress(raw string) (string, string, error) {
	host, port, found := strings.Cut(raw, ":")
	if !found {
		return "", "", fmt.Errorf("address %q has no port", raw)
	}
	return host, port, nil
}
