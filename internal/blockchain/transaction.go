package blockchain

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/rand"
	"crypto/sha256"
	"encoding/binary"
	"encoding/gob"
	"log"
	"math/big"
	"time"
)

type TransactionType uint8

const (
	REGISTER TransactionType = iota
	UPDATE
	REVOKE
)

type Transaction struct {
	TID        int
	Type       TransactionType
	Timestamp  int64
	DomainName string
	IP         string
	TTL        int64
	OwnerKey   []byte
	Signature  []byte
}

func NewTransaction(txType TransactionType, domainName, ip string, ttl int64, ownerKey []byte,
	privateKey *ecdsa.PrivateKey, txPool map[int]*Transaction) *Transaction {
	tx := Transaction{
		TID:        GenerateRandomTxID(txPool),
		Type:       txType,
		Timestamp:  time.Now().Unix(),
		DomainName: domainName,
		IP:         ip,
		TTL:        ttl,
		OwnerKey:   ownerKey,
		Signature:  nil,
	}

	tx.Signature = SignTransaction(privateKey, &tx)
	return &tx
}

func SignTransaction(privateKey *ecdsa.PrivateKey, tx *Transaction) []byte {
	txData := tx.SerializeForSigning()
	hash := sha256.Sum256(txData)

	r, s, err := ecdsa.Sign(rand.Reader, privateKey, hash[:])
	if err != nil {
		log.Panic("Failed to sign transaction:", err)
		return nil
	}

	// Ensure r and s are exactly 32 bytes each
	rBytes := r.Bytes()
	sBytes := s.Bytes()

	// Pad r and s to fixed 32-byte size
	rPadded := make([]byte, 32)
	sPadded := make([]byte, 32)
	copy(rPadded[32-len(rBytes):], rBytes)
	copy(sPadded[32-len(sBytes):], sBytes)

	// Concatenate r and s
	signature := append(rPadded, sPadded...)
	tx.Signature = signature

	return signature
}

func VerifyTransaction(publicKeyBytes []byte, tx *Transaction) bool {
	publicKey, err := BytesToPublicKey(publicKeyBytes)
	if err != nil {
		log.Println("Invalid public key format")
		return false
	}

	txData := tx.SerializeForSigning()
	hash := sha256.Sum256(txData)

	// Ensure signature length is correct
	if len(tx.Signature) != 64 {
		log.Println("Invalid signature length")
		return false
	}

	// Extract r and s from the signature
	r := new(big.Int).SetBytes(tx.Signature[:32])
	s := new(big.Int).SetBytes(tx.Signature[32:])

	return ecdsa.Verify(publicKey, hash[:], r, s)
}

func (tx *Transaction) SerializeForSigning() []byte {
	txData := append(IntToByteArr(int64(tx.TID)), byte(tx.Type))
	txData = append(txData, IntToByteArr(tx.Timestamp)...)
	txData = append(txData, []byte(tx.DomainName)...)
	txData = append(txData, []byte(tx.IP)...)
	txData = append(txData, IntToByteArr(tx.TTL)...)

	return txData
}

func (tx *Transaction) Serialize() []byte {
	var result bytes.Buffer
	encoder := gob.NewEncoder(&result)

	err := encoder.Encode(tx)
	if err != nil {
		log.Panic(err)
	}

	return result.Bytes()
}

func DeserializeTx(d []byte) *Transaction {
	var transaction Transaction

	decoder := gob.NewDecoder(bytes.NewReader(d))
	err := decoder.Decode(&transaction)
	if err != nil {
		log.Panic(err)
	}

	return &transaction
}

// TODO: Check if TxID is among existing IDs, not just in the pool
// GenerateRandomTxID generates a unique random transaction ID using crypto/rand
func GenerateRandomTxID(txPool map[int]*Transaction) int {
	for {
		var buf [8]byte
		_, err := rand.Read(buf[:]) // Read 8 random bytes
		if err != nil {
			panic("Failed to generate random transaction ID")
		}

		txID := int(binary.LittleEndian.Uint64(buf[:]) % 1_000_000_000) // Ensure within range

		if _, exists := txPool[txID]; !exists { // Ensure uniqueness
			return txID
		}
	}
}

func RemoveTxsFromPool(txs []Transaction, txPool map[int]*Transaction) {
	for _, tx := range txs {
		delete(txPool, tx.TID)
	}
}
