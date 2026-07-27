package scenario

import "errors"

var (
	ErrNegativeSize = errors.New("block size cannot be negative")
	ErrOverCapacity = errors.New("blocks do not fit within the capacity")
)

func Offsets(sizes []int, capacity int) ([]int, error) {
	total := 0
	for _, size := range sizes {
		if size < 0 {
			return nil, ErrNegativeSize
		}
		total += size
	}
	if total > capacity {
		return nil, ErrOverCapacity
	}

	offsets := make([]int, 0, len(sizes))
	next := 0
	for _, size := range sizes {
		if next+size > capacity {
			panic("offsets exceeded capacity after the total was checked")
		}
		offsets = append(offsets, next)
		next += size
	}
	return offsets, nil
}
