package blockchain

import (
	"bytes"
	"crypto/sha256"
	"encoding/gob"
	"encoding/hex"
	"log"
	"strconv"
	"time"
)

// RegistrationRecord stores details for domain registration
type RegistrationRecord struct {
	TID              string
	Timestamp        int64
	DomainName       string
	IP               string
	TTL              int64
	OwnerKey         string
	RegistrySignature string
}

// UpdateRecord stores details for updating an existing domain record
type UpdateRecord struct {
	TID           string
	Timestamp     int64
	DomainName    string
	IP            string
	OwnerKey      string
	OwnerSignature string
}

// RevocationRecord stores details for revoking a domain record
type RevocationRecord struct {
	TID        string
	Timestamp  int64
	DomainName string
	Signature  string
	Key        string
}

// Block represents a single block in the B-DNS blockchain
type Block struct {
	Index               int
	PreviousHash        string
	Timestamp           int64
	SlotLeader          string
	State               string
	Signature           string
	RegistrationRecords []RegistrationRecord
	UpdateRecords       []UpdateRecord
	RevocationRecords   []RevocationRecord
	Hash                string
}

// ComputeHash generates a hash for the block
func (b *Block) ComputeHash() string {
	recordData := ""
	for _, r := range b.RegistrationRecords {
		recordData += r.TID + r.DomainName + r.IP + r.OwnerKey + r.RegistrySignature
	}
	for _, u := range b.UpdateRecords {
		recordData += u.TID + u.DomainName + u.IP + u.OwnerKey + u.OwnerSignature
	}
	for _, rev := range b.RevocationRecords {
		recordData += rev.TID + rev.DomainName + rev.Signature + rev.Key
	}

	data := strconv.Itoa(b.Index) + b.PreviousHash + strconv.FormatInt(b.Timestamp, 10) +
		b.SlotLeader + b.State + b.Signature + recordData

	hash := sha256.Sum256([]byte(data))
	return hex.EncodeToString(hash[:])
}

// NewBlock creates a new block given previous block's hash and operation records
func NewBlock(index int, previousHash string, slotLeader string, state string, signature string,
	reg []RegistrationRecord, upd []UpdateRecord, rev []RevocationRecord) *Block {

	block := &Block{
		Index:               index,
		PreviousHash:        previousHash,
		Timestamp:           time.Now().Unix(),
		SlotLeader:          slotLeader,
		State:               state,
		Signature:           signature,
		RegistrationRecords: reg,
		UpdateRecords:       upd,
		RevocationRecords:   rev,
	}
	block.Hash = block.ComputeHash()
	return block
}

// NewGenesisBlock creates the genesis block with initial registry list and randomness
func NewGenesisBlock() *Block {
	genesisBlock := &Block{
		Index:        0,
		PreviousHash: "",
		Timestamp:    time.Now().Unix(),
		SlotLeader:   "genesis_leader",
		State:        "genesis_state",
		Signature:    "genesis_signature",
	}
	genesisBlock.Hash = genesisBlock.ComputeHash()
	return genesisBlock
}

// SerializeBlock serializes a block into bytes
func SerializeBlock(b *Block) []byte {
	var result bytes.Buffer
	encoder := gob.NewEncoder(&result)

	err := encoder.Encode(b)
	if err != nil {
		log.Panic(err)
	}

	return result.Bytes()
}

// DeserializeBlock deserializes a block from bytes
func DeserializeBlock(d []byte) *Block {
	var block Block

	decoder := gob.NewDecoder(bytes.NewReader(d))
	err := decoder.Decode(&block)
	if err != nil {
		log.Panic(err)
	}

	return &block
}
