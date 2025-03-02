package network

import (
	"fmt"
	"net"
	"strings"

	// "github.com/bleasey/bdns/internal/blockchain"
)

// ResolveDomain queries the BDNS blockchain for a domain
func ResolveDomain(domain string) (string, error) {
	// Check local cache first
	if ip, found := GetFromCache(domain); found {
		return ip, nil
	}

	// Query blockchain if it’s a BDNS domain
	if strings.HasSuffix(domain, ".bdns.") {
		record, found := "127.0.0.1:52670", true // Query blockchain for domain
		if !found {
			return "", fmt.Errorf("domain %s not found in BDNS", domain)
		}

		// Cache the resolved domain for faster lookups
		SetToCache(domain, record)
		return record, nil
	}

	// Default to traditional DNS resolution
	ips, err := net.LookupHost(domain)
	if err != nil {
		return "", fmt.Errorf("DNS resolution failed: %v", err)
	}

	return ips[0], nil // Return first resolved IP
}
