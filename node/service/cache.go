package service

import (
	"sync"
)

type compiledCache struct {
	sync.RWMutex
	cache map[uint64]*compiled
}

func newCache() *compiledCache {
	return &compiledCache{
		cache: make(map[uint64]*compiled),
	}
}

func (cc *compiledCache) Get(id uint64) *compiled {
	cc.RLock()
	defer cc.RUnlock()
	return cc.cache[id]
}

func (cc *compiledCache) Add(id uint64, c *compiled) {
	cc.Lock()
	defer cc.Unlock()
	cc.cache[id] = c
}

func (cc *compiledCache) Del(id uint64) {
	cc.Lock()
	defer cc.Unlock()
	delete(cc.cache, id)
}
