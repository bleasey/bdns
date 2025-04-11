package network

import (
	"fmt"
	"net"
	"strings"
	"time"
	// "github.com/bleasey/bdns/internal/blockchain"
)

// ResolveDomain queries the BDNS blockchain for a domain
func ResolveDomain(domain string, n *Node) (string, error) {
	// Check local cache first
	if ip, found := GetFromCache(domain); found {
		return ip, nil
	}

	/* Query blockchain if it’s a BDNS domain
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

	return ips[0], nil // Return first resolved IP */
	// 2. BDNS record lookup if it ends with ".bdns."
	if strings.HasSuffix(domain, ".bdns.") {
		n.TxMutex.Lock()
		tx := n.IndexManager.GetIP(domain)
		n.TxMutex.Unlock()

		if tx != nil {
			SetToCache(domain, tx.IP)
			return tx.IP, nil
		}

		// Light node fallback: send query to peers
		if tx == nil && !n.IsFullNode {
			n.MakeDNSRequest(domain) // light node forwarding
			time.Sleep(2 * time.Second)
			tx = n.IndexManager.GetIP(domain)
		}

		n.TxMutex.Lock()
		tx = n.IndexManager.GetIP(domain)
		n.TxMutex.Unlock()
		if tx != nil {
			SetToCache(domain, tx.IP)
			return tx.IP, nil
		}
		return "", fmt.Errorf("domain %s not found in B-DNS", domain)
	}

	// 3. Legacy DNS fallback
	ips, err := net.LookupHost(domain)
	if err != nil || len(ips) == 0 {
		return "", fmt.Errorf("legacy DNS resolution failed: %v", err)
	}

	SetToCache(domain, ips[0])
	return ips[0], nil
}
