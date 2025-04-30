package container

import "sync"

type c[T any] struct {
	sync.Mutex
	value T
}

type Container[T any] interface {
	ReplaceWith(item T)
	GetLatest() T
}

func (c *c[T]) ReplaceWith(item T) {
	c.Lock()
	defer c.Unlock()
	c.value = item
}

func (c *c[T]) GetLatest() T {
	c.Lock()
	defer c.Unlock()

	return c.value
}

func New[T any](initialItem T) Container[T] {
	return &c[T]{value: initialItem}
}
