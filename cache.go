package evac

import (
	"github.com/miekg/dns"
	"sync"
)

type Cache struct {
	internal_cache map[string]*dns.RR
	lock           sync.RWMutex
}

func NewCache() *Cache {
	return &Cache{internal_cache: make(map[string]*dns.RR),
		lock: sync.RWMutex{}}
}

func (cache *Cache) GetRecord(domain string) (*dns.RR, bool) {
	locker := cache.lock.RLocker()
	locker.Lock()
	record, ok := cache.internal_cache[domain]
	locker.Unlock()
	return record, ok
}

func (cache *Cache) UpdateRecord(domain string, record *dns.RR) {
	cache.lock.Lock()
	cache.internal_cache[domain] = record
	cache.lock.Unlock()
}
