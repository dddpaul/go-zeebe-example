package cache

import "sync"

var (
	cache map[string]chan bool
	mu    sync.Mutex
)

func init() {
	cache = make(map[string]chan bool)
}

func Add(uuid string, ch chan bool) {
	mu.Lock()
	cache[uuid] = ch
	defer mu.Unlock()
}

func Get(uuid string) chan bool {
	return cache[uuid]
}

func Del(uuid string) {
	mu.Lock()
	delete(cache, uuid)
	defer mu.Unlock()
}
