package cache

import (
	"sync"
	"time"
)

var (
	cache      = make(map[string]cacheEntry)
	cacheMutex sync.Mutex
)

// cacheEntry holds DNS records with TTL
type cacheEntry struct {
	ip       string
	expiry   time.Time
}

// Get retrieves a domain from the cache
func Get(domain string) (string, bool) {
	cacheMutex.Lock()
	defer cacheMutex.Unlock()

	entry, found := cache[domain]
	if !found || time.Now().After(entry.expiry) {
		delete(cache, domain) // Clean up expired entries
		return "", false
	}
	return entry.ip, true
}

// Set stores a domain in the cache with a default TTL of 300 seconds
func Set(domain, ip string) {
	cacheMutex.Lock()
	defer cacheMutex.Unlock()

	cache[domain] = cacheEntry{
		ip:     ip,
		expiry: time.Now().Add(300 * time.Second), // 5-minute TTL
	}
}
