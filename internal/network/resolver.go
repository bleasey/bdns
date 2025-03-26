package network

import (
	"fmt"
	"net"
	
)

// ResolveDomain converts legacy DNS queries to BDNS and processes them
func ResolveDomain(domain string) (string, error) {
	// Check local cache first
	if ip, found := GetFromCache(domain); found {
		return ip, nil
	}

	// Convert legacy DNS queries into BDNS-style lookups
	bdnsDomain := ConvertToBDNS(domain)

	// Query BDNS first
	record, found := QueryBDNS(bdnsDomain)
	if found {
		SetToCache(domain, record) // Store original domain in cache
		return record, nil
	}

	// If BDNS fails, fallback to legacy DNS resolution
	return ResolveLegacyDNS(domain)
}

// ConvertToBDNS changes standard domain queries into BDNS-compatible format
func ConvertToBDNS(domain string) string {
	// Example: "example.com" → "example.com.bdns."
	return domain + ".bdns."
}

// QueryBDNS tries to resolve the BDNS domain
func QueryBDNS(domain string) (string, bool) {
	// Simulating a blockchain query
	record := "127.0.0.1" // Replace with blockchain query
	return record, true
}

// ResolveLegacyDNS performs standard DNS resolution for non-BDNS domains
func ResolveLegacyDNS(domain string) (string, error) {
	ips, err := net.LookupHost(domain)
	if err != nil {
		return "", fmt.Errorf("Legacy DNS resolution failed: %v", err)
	}
	return ips[0], nil // Return the first resolved IP
}
