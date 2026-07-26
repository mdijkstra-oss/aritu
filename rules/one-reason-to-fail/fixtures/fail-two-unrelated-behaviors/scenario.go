package scenario

import (
	"errors"
	"strings"
)

func NormalizeUsername(raw string) (string, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return "", errors.New("username must not be blank")
	}
	return strings.ToLower(trimmed), nil
}
