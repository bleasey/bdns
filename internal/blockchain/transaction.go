package blockchain

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/rand"
	"crypto/sha256"
	"math/big"
	"encoding/gob"
	"log"
)

type TransactionType uint8

const (
	REGISTER TransactionType = iota
	UPDATE
	REVOKE
)

type Transaction struct {
	TID        []byte
	Type       TransactionType
	Timestamp  int64
	DomainName string
	IP         string
	TTL        int64
	OwnerKey   []byte
	Signature  []byte
}

func SignTransaction(privateKey *ecdsa.PrivateKey, tx *Transaction) []byte {
	// Serialize and hash the transaction data
	txData := tx.SerializeForSigning()
	hash := sha256.Sum256(txData)

	// Sign the hash
	r, s, err := ecdsa.Sign(rand.Reader, privateKey, hash[:])
	if err != nil {
		log.Panic(err)
		return nil
	}

	// Serialize r and s into a signature
	signature := append(r.Bytes(), s.Bytes()...)
	return signature
}

func VerifyTransaction(publicKey *ecdsa.PublicKey, tx *Transaction) bool {
	// Serialize and hash the transaction data
	txData := tx.SerializeForSigning()
	hash := sha256.Sum256(txData)

	// Extract r and s from the signature
	signature := tx.Signature
	if len(signature) < 64 {
		return false // Invalid signature length
	}

	r := new(big.Int).SetBytes(signature[:32])
	s := new(big.Int).SetBytes(signature[32:])

	// Verify the signature
	return ecdsa.Verify(publicKey, hash[:], r, s)
}

func (tx *Transaction) SerializeForSigning() []byte {
	txData := append(tx.TID, byte(tx.Type))
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
