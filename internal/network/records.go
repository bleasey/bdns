package network

import "time"

// RecordType defines supported BDNS record types
type RecordType string

const (
	A     RecordType = "A"     // IPv4 Address
	AAAA  RecordType = "AAAA"  // IPv6 Address
	CNAME RecordType = "CNAME" // Alias for another domain
	MX    RecordType = "MX"    // Mail Exchange
)

// DNSRecord represents a BDNS record stored in the blockchain
type DNSRecord struct {
	DomainName string      `json:"domain"`   // The domain being registered
	RecordType RecordType  `json:"type"`     // The type of DNS record
	Value      string      `json:"value"`    // IP, alias, mail server, or other value
	TTL        int64       `json:"ttl"`      // Time-To-Live in seconds
	Priority   int         `json:"priority"` // Used for MX/SRV records
	Port       int         `json:"port"`     // Used for SRV records
	OwnerKey   string      `json:"ownerKey"` // Owner's public key
	Signature  string      `json:"signature"`// Signature to verify authenticity
	Timestamp  time.Time   `json:"timestamp"`// Record creation time
}
