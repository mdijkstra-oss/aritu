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

func Parse(raw string) (Address, error) {
	colon := strings.LastIndex(raw, ":")
	if colon < 0 {
		return Address{}, fmt.Errorf("address %q has no port", raw)
	}

	host, digits := raw[:colon], raw[colon+1:]
	if host == "" {
		return Address{}, fmt.Errorf("address %q has an empty host", raw)
	}

	port, err := strconv.Atoi(digits)
	if err != nil {
		return Address{}, fmt.Errorf("port %q is not a number", digits)
	}
	if port < minPort || port > maxPort {
		return Address{}, fmt.Errorf("port %d is out of range %d-%d", port, minPort, maxPort)
	}

	return Address{Host: host, Port: port}, nil
}

const (
	minPort = 1
	maxPort = 65535
)
