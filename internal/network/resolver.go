package network

import (
	"fmt"
	"time"
)

// ResolveDomain queries the BDNS blockchain for a domain
func ResolveDomain(domain string, n *Node) (string, error) {
	// 1. Check local cache
	if ip, found := GetFromCache(domain); found {
		return ip, nil
	}

	// 2. Check local blockchain state
	n.TxMutex.Lock()
	tx := n.IndexManager.GetIP(domain)
	n.TxMutex.Unlock()

	if tx != nil {
		SetToCache(domain, tx.IP)
		return tx.IP, nil
	}

	// 3. If light node, forward DNS query to peers
	if !n.IsFullNode {
		n.MakeDNSRequest(domain)
		time.Sleep(2 * time.Second)

		n.TxMutex.Lock()
		tx = n.IndexManager.GetIP(domain)
		n.TxMutex.Unlock()

		if tx != nil {
			SetToCache(domain, tx.IP)
			return tx.IP, nil
		}
	}

	// 4. Not found in BDNS
	return "", fmt.Errorf("domain %s not found in B-DNS", domain)
}