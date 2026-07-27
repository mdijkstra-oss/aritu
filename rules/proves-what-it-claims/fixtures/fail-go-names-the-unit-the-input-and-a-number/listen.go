package scenario

import (
	"fmt"
	"strconv"
	"strings"
)

type Address struct {
	Host string
	Port int
}

const wildcardHost = "0.0.0.0"

func ParseListenAddress(raw string) (Address, error) {
	colon := strings.LastIndex(raw, ":")
	if colon < 0 {
		return Address{}, fmt.Errorf("listen address %q has no port", raw)
	}

	port, err := strconv.Atoi(raw[colon+1:])
	if err != nil {
		return Address{}, fmt.Errorf("listen address %q has a non-numeric port: %w", raw, err)
	}
	if port < 1 || port > 65535 {
		return Address{}, fmt.Errorf("listen address %q has port %d outside 1-65535", raw, port)
	}

	host := raw[:colon]
	if host == "" {
		host = wildcardHost
	}
	return Address{Host: host, Port: port}, nil
}
