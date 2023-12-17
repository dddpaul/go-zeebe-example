package cache

import "sync"

var (
	cache map[string]chan bool
	mu    sync.Mutex
)

func init() {
	cache = make(map[string]chan bool)
}

func Add(id string, ch chan bool) {
	mu.Lock()
	cache[id] = ch
	defer mu.Unlock()
}

func Get(id string) chan bool {
	return cache[id]
}

func Del(id string) {
	mu.Lock()
	delete(cache, id)
	defer mu.Unlock()
}
