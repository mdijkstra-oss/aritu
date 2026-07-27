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
	port, err := strconv.Atoi(raw[colon+1:])
	if err != nil {
		return Address{}, fmt.Errorf("address %q has a non-numeric port: %w", raw, err)
	}
	return Address{Host: raw[:colon], Port: port}, nil
}
