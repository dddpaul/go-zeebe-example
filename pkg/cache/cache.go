package cache

import "sync"

var (
	cache map[string]chan interface{}
	mu    sync.Mutex
)

func init() {
	cache = make(map[string]chan interface{})
}

func Add(id string) (chan interface{}, func(id string)) {
	ch := make(chan interface{}, 1)
	mu.Lock()
	cache[id] = ch
	defer mu.Unlock()
	return ch, Del
}

func Get(id string) chan interface{} {
	return cache[id]
}

func Del(id string) {
	mu.Lock()
	delete(cache, id)
	defer mu.Unlock()
}
