package network

import (
	"fmt"
	"net"
	"strings"
	"time"
)

// ResolveDomain queries the BDNS blockchain for a domain
func ResolveDomain(domain string, n *Node) (string, error) {
	fmt.Printf("Resolving domain: %s\n", domain)

	// Check local cache first
	if ip, found := GetFromCache(domain); found {
		return ip, nil
	}

	if strings.HasSuffix(domain, ".bdns.") {
		n.TxMutex.Lock()
		tx := n.IndexManager.GetIP(domain)
		n.TxMutex.Unlock()

		if tx != nil {
			SetToCache(domain, tx.IP)
			return tx.IP, nil
		}

		// Light node fallback: send query to peers
		if !n.IsFullNode {
			n.MakeDNSRequest(domain, nil) // light node forwarding
			time.Sleep(2 * time.Second)
			tx = n.IndexManager.GetIP(domain)
			if tx != nil {
				SetToCache(domain, tx.IP)
				return tx.IP, nil
			}
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
