package scenario

import (
	"fmt"
	"strconv"
)

const maxPort = 65535

func ParsePort(raw string) (int, error) {
	port, err := strconv.Atoi(raw)
	if err != nil {
		return 0, fmt.Errorf("port %q is not a number", raw)
	}
	if port < 1 || port > maxPort {
		return 0, fmt.Errorf("port %d is outside 1-%d", port, maxPort)
	}
	return port, nil
}
