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

func ParseAddress(raw string) (Address, error) {
	host, portText, isSplit := strings.Cut(raw, ":")
	if !isSplit {
		return Address{}, fmt.Errorf("address %q: want host:port", raw)
	}
	port, err := strconv.Atoi(portText)
	if err != nil {
		return Address{}, fmt.Errorf("address %q: %w", raw, err)
	}
	return Address{Host: host, Port: port}, nil
}
