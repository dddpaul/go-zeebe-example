package cache

import (
	"sync"
)

var cache Cache

type Cache interface {
	Add(string) (chan interface{}, func(string))
	Get(string) chan interface{}
	Del(string)
}

type SimpleCache struct {
	cache map[string]chan interface{}
	mu    sync.Mutex
}

func init() {
	cache = NewSimpleCache()
}

func Add(id string) (chan interface{}, func(id string)) {
	return cache.Add(id)
}

func Get(id string) chan interface{} {
	return cache.Get(id)
}

func Del(id string) {
	cache.Del(id)
}

func NewSimpleCache() Cache {
	return &SimpleCache{
		cache: make(map[string]chan interface{}),
	}
}
func (c *SimpleCache) Add(id string) (chan interface{}, func(id string)) {
	ch := make(chan interface{}, 1)
	c.mu.Lock()
	c.cache[id] = ch
	defer c.mu.Unlock()
	return ch, c.Del
}

func (c *SimpleCache) Get(id string) chan interface{} {
	return c.cache[id]
}

func (c *SimpleCache) Del(id string) {
	c.mu.Lock()
	delete(c.cache, id)
	defer c.mu.Unlock()
}
