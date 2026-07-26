package scenario

import (
	"fmt"
	"strconv"
	"strings"
)

type Version struct {
	Major int
	Minor int
	Patch int
}

func ParseSemver(raw string) (Version, error) {
	parts := strings.Split(raw, ".")
	if len(parts) != 3 {
		return Version{}, fmt.Errorf("version %q: want major.minor.patch", raw)
	}

	numbers := make([]int, 0, len(parts))
	for _, part := range parts {
		number, err := strconv.Atoi(part)
		if err != nil {
			return Version{}, fmt.Errorf("version %q: %q is not a number", raw, part)
		}
		numbers = append(numbers, number)
	}

	return Version{Major: numbers[0], Minor: numbers[1], Patch: numbers[2]}, nil
}
