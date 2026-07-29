package pricing

import "sync"

type Quote struct {
	Symbol string
	Cents  int
}

func QuoteFor(symbol string, rates *rateCache, fetch func(string) (int, error)) (Quote, error) {
	cents, err := rates.once(symbol, func() (int, error) { return fetch(symbol) })
	if err != nil {
		return Quote{}, err
	}
	return Quote{Symbol: symbol, Cents: cents}, nil
}

type rateCache struct {
	mu      sync.Mutex
	entries map[string]*rateEntry
}

type rateEntry struct {
	once  sync.Once
	value int
	err   error
}

func newRateCache() *rateCache {
	return &rateCache{entries: map[string]*rateEntry{}}
}

func (c *rateCache) once(key string, compute func() (int, error)) (int, error) {
	entry := c.entryFor(key)
	entry.once.Do(func() {
		entry.value, entry.err = compute()
	})
	return entry.value, entry.err
}

func (c *rateCache) entryFor(key string) *rateEntry {
	c.mu.Lock()
	defer c.mu.Unlock()
	entry, claimed := c.entries[key]
	if !claimed {
		entry = &rateEntry{}
		c.entries[key] = entry
	}
	return entry
}
