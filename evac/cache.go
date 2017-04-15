package evac

import (
	"github.com/miekg/dns"
	"sync"
	"time"
)

type dnsCacheEntry struct {
	record dns.RR
	time_added time.Time
}

type dnsRecordMap map[uint16]dnsCacheEntry
type dnsCacheMap map[string]dnsRecordMap

type Cache struct {
	internal_cache dnsCacheMap
	lock           sync.RWMutex
	limit          uint32
	count          uint32
}

func NewCache(limit uint32) *Cache {
	return &Cache{internal_cache: make(dnsCacheMap),
		lock: sync.RWMutex{}, limit: limit, count: 0}
}

func (cache *Cache) GetRecord(domain string, dnstype uint16) (dns.RR, bool) {
	locker := cache.lock.RLocker()
	locker.Lock()
	var entry dnsCacheEntry = nil
	var found bool = false
	if record_map, ok := cache.internal_cache[domain]; ok {
		entry, found = record_map[dnstype]
	}
	locker.Unlock()
	return &entry.record, found
}

func (cache *Cache) UpdateRecord(domain string, record dns.RR) {
	header := record.Header()
	cache.lock.Lock()
	if _, ok := cache.internal_cache[domain]; !ok {
		cache.internal_cache[domain] = make(dnsRecordMap)
	}
	_, ok := cache.internal_cache[domain][header.Rrtype]
	if !ok {
		// Check if we have to do a random replacement
		if cache.count >= cache.limit {
			cache.performRandomCacheReplacement()
		}
	}
	cache.internal_cache[domain][header.Rrtype] = dnsCacheEntry{ record: record,
	time_added: time.Now()}
	cache.lock.Unlock()
}

func (cache *Cache) TTLExpirationCleanup() {
	cache.lock.Lock()
	for domain, records := range cache.internal_cache {
		for record_type, record := range records {
			duration_since := time.Since(record.time_added)
			dns_record_header := record.record.Header()

			if duration_since.Seconds() > float64(dns_record_header.Ttl) {
				cache.deleteRecordNotLocked(domain, record_type)
			}
		}
	}
	cache.lock.Unlock()
}

func (cache *Cache) deleteRecordNotLocked(domain string, record_type uint16) {
	delete(cache.internal_cache[domain], record_type)
	cache.count -= 1
}

func (cache *Cache) performRandomCacheReplacement() {
	for domain, records := range cache.internal_cache {
		for record_type, _ := range records {
			cache.deleteRecordNotLocked(domain, record_type)
			return
		}
	}
}