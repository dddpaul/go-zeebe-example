package cache

import "sync"

var (
	cache map[string]chan bool
	mu    sync.Mutex
)

func init() {
	cache = make(map[string]chan bool)
}

func Add(id string) (chan bool, func(id string)) {
	ch := make(chan bool, 1)
	mu.Lock()
	cache[id] = ch
	defer mu.Unlock()
	return ch, Del
}

func Get(id string) chan bool {
	return cache[id]
}

func Del(id string) {
	mu.Lock()
	delete(cache, id)
	defer mu.Unlock()
}
