package evac

import (
	"github.com/miekg/dns"
	"sync"
)

type dnsRecordMap map[uint16]dns.RR

type dnsCacheMap map[string]dnsRecordMap

type Cache struct {
	internal_cache dnsCacheMap
	lock           sync.RWMutex
}

func NewCache() *Cache {
	return &Cache{internal_cache: make(dnsCacheMap),
		lock: sync.RWMutex{}}
}

func (cache *Cache) GetRecord(domain string, dnstype uint16) (*dns.RR, bool) {
	locker := cache.lock.RLocker()
	locker.Lock()
	var record dns.RR = nil
	var found bool = false
	if recordmap, ok := cache.internal_cache[domain]; ok {
		record, found = recordmap[dnstype]
	}
	locker.Unlock()
	return &record, found
}

func (cache *Cache) UpdateRecord(domain string, record dns.RR) {
	cache.lock.Lock()
	header := record.Header()
	if _, ok := cache.internal_cache[domain]; !ok {
		cache.internal_cache[domain] = make(dnsRecordMap)
	}
	cache.internal_cache[domain][header.Rrtype] = record
	cache.lock.Unlock()
}
