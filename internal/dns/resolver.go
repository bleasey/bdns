package dns

import (
	"fmt"
	"log"
	"net"
	"strings"
	"bdns/blockchain"
	"bdns/dns/cache"
)

// ResolveDomain queries the BDNS blockchain for a domain
func ResolveDomain(domain string) (string, error) {
	// Check local cache first
	if ip, found := cache.Get(domain); found {
		return ip, nil
	}

	// Query blockchain if it’s a BDNS domain
	if strings.HasSuffix(domain, ".bdns.") {
		record, found := blockchain.BDNSChain.GetDomainRecord(domain)
		if !found {
			return "", fmt.Errorf("domain %s not found in BDNS", domain)
		}

		// Cache the resolved domain for faster lookups
		cache.Set(domain, record.IP)
		return record.IP, nil
	}

	// Default to traditional DNS resolution
	ips, err := net.LookupHost(domain)
	if err != nil {
		return "", fmt.Errorf("DNS resolution failed: %v", err)
	}

	return ips[0], nil // Return first resolved IP
}
