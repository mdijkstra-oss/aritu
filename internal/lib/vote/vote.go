package vote

import (
	"context"
	"errors"
	"fmt"
	"sync"
)

func Collect[T any](ctx context.Context, n int, fn func(ctx context.Context, round int) (T, error)) ([]T, error) {
	if n <= 0 {
		return nil, fmt.Errorf("%w: got %d", errNoRounds, n)
	}

	roundCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	results := make([]T, n)
	var (
		wg    sync.WaitGroup
		once  sync.Once
		first error
	)
	for round := range n {
		wg.Add(1)
		go func() {
			defer wg.Done()
			result, err := fn(roundCtx, round)
			if err != nil {
				once.Do(func() {
					first = err
					cancel()
				})
				return
			}
			results[round] = result
		}()
	}
	wg.Wait()

	if first != nil {
		return nil, first
	}
	return results, nil
}

func Tally[K comparable](rounds []map[K]bool) map[K]int {
	counts := make(map[K]int)
	for _, round := range rounds {
		for key, isSatisfied := range round {
			count := counts[key]
			if isSatisfied {
				count++
			}
			counts[key] = count
		}
	}
	return counts
}

func IsUnanimous[K comparable](counts map[K]int, total int) bool {
	for _, count := range counts {
		if count != total {
			return false
		}
	}
	return true
}

var errNoRounds = errors.New("vote: rounds must be positive")
