package processing

import (
	"github.com/miekg/dns"
	"time"
	"sync"
	"fmt"
)

type dnsCacheEntry struct {
	records    []dns.RR
	time_added time.Time
	is_blocked bool
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

func (cache *Cache) GetRecord(domain string, dnstype uint16) ([]dns.RR, bool, bool) {
	locker := cache.lock.RLocker()
	locker.Lock()
	defer locker.Unlock()

	var entry dnsCacheEntry
	var found bool = false
	if record_map, ok := cache.internal_cache[domain]; ok {
		entry, found = record_map[dnstype]
	}
	return entry.records, found, entry.is_blocked
}

func (cache *Cache) UpdateBlockedRecord(domain string, record_type uint16) {
	cache.lock.Lock()
	defer cache.lock.Unlock()

	cache.prepareDnsRecordMap(domain, record_type)
	cache.internal_cache[domain][record_type] = dnsCacheEntry{
		records: nil,
		time_added: time.Now(),
		is_blocked: true,
	}
}

func (cache *Cache) UpdateRecord(domain string, queryType uint16, records []dns.RR) {
	if len(records) < 1 {
		fmt.Printf("Tried to set empty records for domain '%s'", domain)
		return
	}
	cache.lock.Lock()
	defer cache.lock.Unlock()

	cache.prepareDnsRecordMap(domain, queryType)
	cache.internal_cache[domain][queryType] = dnsCacheEntry{
		records: records,
		time_added: time.Now(),
		is_blocked: false,
	}
}

func (cache *Cache) TTLExpirationCleanup() {
	cache.lock.Lock()
	defer cache.lock.Unlock()

	for domain, records := range cache.internal_cache {
		for record_type, record := range records {
			duration_since := time.Since(record.time_added)
			dns_record_header := record.records[0].Header()

			if duration_since.Seconds() > float64(dns_record_header.Ttl) {
				cache.deleteRecordNotLocked(domain, record_type)
			}
		}
	}
}

func (cache *Cache) Flush() {
	cache.lock.Lock()
	defer cache.lock.Unlock()

	for domain, records := range cache.internal_cache {
		for record_type, _ := range records {
			cache.deleteRecordNotLocked(domain, record_type)
		}
	}
}

func (cache *Cache) prepareDnsRecordMap(domain string, record_type uint16) {
	if _, ok := cache.internal_cache[domain]; !ok {
		cache.internal_cache[domain] = make(dnsRecordMap)
	}
	_, ok := cache.internal_cache[domain][record_type]
	if !ok {
		/* Check if we have to do a random replacement */
		if cache.count >= cache.limit {
			cache.performRandomCacheReplacement()
		}
	}
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
