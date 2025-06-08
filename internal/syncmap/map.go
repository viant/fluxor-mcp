package syncmap

import "sync"

// Map is a thread-safe generic map structure
type Map[T any] struct {
	mux sync.RWMutex
	m   map[string]T
}

// NewRegistry creates a new instance of Map
func NewRegistry[T any]() *Map[T] {
	return &Map[T]{
		m: make(map[string]T),
	}
}

// Get retrieves an item by name
func (r *Map[T]) Get(name string) T {
	r.mux.RLock()
	defer r.mux.RUnlock()
	if v, ok := r.m[name]; ok {
		return v
	}
	var zero T
	return zero
}

// Set adds or updates an item by name
func (r *Map[T]) Set(name string, value T) {
	r.mux.Lock()
	defer r.mux.Unlock()
	r.m[name] = value
}

// Delete removes an item by name
func (r *Map[T]) Delete(name string) {
	r.mux.Lock()
	defer r.mux.Unlock()
	delete(r.m, name)
}

// List returns a slice of all items
func (r *Map[T]) List() []T {
	r.mux.RLock()
	defer r.mux.RUnlock()
	ret := make([]T, 0, len(r.m))
	for _, v := range r.m {
		ret = append(ret, v)
	}
	return ret
}
