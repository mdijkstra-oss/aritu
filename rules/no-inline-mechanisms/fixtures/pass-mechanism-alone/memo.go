package memo

import "sync"

type Map[K comparable, V any] struct {
	mu    sync.Mutex
	cells map[K]*cell[V]
}

type cell[V any] struct {
	once  sync.Once
	value V
	err   error
}

func NewMap[K comparable, V any]() *Map[K, V] {
	return &Map[K, V]{cells: map[K]*cell[V]{}}
}

func (m *Map[K, V]) Once(key K, compute func() (V, error)) (V, error) {
	found := m.cellFor(key)
	found.once.Do(func() {
		found.value, found.err = compute()
	})
	return found.value, found.err
}

func (m *Map[K, V]) cellFor(key K) *cell[V] {
	m.mu.Lock()
	defer m.mu.Unlock()
	found, claimed := m.cells[key]
	if !claimed {
		found = &cell[V]{}
		m.cells[key] = found
	}
	return found
}
