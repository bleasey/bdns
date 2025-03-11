package blockchain

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/x509"
	"encoding/binary"
	"log"
	"os"
	"errors"
	"fmt"
)

// Converts an int64 to a byte array
func IntToByteArr(num int64) []byte {
	buff := new(bytes.Buffer)
	err := binary.Write(buff, binary.BigEndian, num)
	if err != nil {
		log.Panic(err)
	}

	return buff.Bytes()
}

func dbExists(dbFile string) bool {
	if _, err := os.Stat(dbFile); os.IsNotExist(err) {
		return false
	}

	return true
}

func BytesToPublicKey(pubKeyBytes []byte) (*ecdsa.PublicKey, error) {
	pubKeyInterface, err := x509.ParsePKIXPublicKey(pubKeyBytes)
	if err != nil {
		return nil, errors.New("invalid public key format")
	}

	pubKey, ok := pubKeyInterface.(*ecdsa.PublicKey)
	if !ok {
		return nil, errors.New("invalid ECDSA public key")
	}

	return pubKey, nil
}

func areStakesEqual(m1, m2 map[string]int) bool {
	if len(m1) != len(m2) {
		return false
	}
	for k, v := range m1 {
		if v2, exists := m2[k]; !exists || v2 != v {
			return false
		}
	}
	return true
}

func (t *Transaction) PrintTx() {
	fmt.Printf("\nTID: %x\n", t.TID)
	fmt.Printf("Type: %d\n", t.Type)
	fmt.Printf("Timestamp: %d\n", t.Timestamp)
	fmt.Printf("DomainName: %s\n", t.DomainName)
	fmt.Printf("IP: %s\n", t.IP)
	fmt.Printf("TTL: %d\n", t.TTL)
	fmt.Printf("OwnerKey: %s\n", t.OwnerKey)
	fmt.Printf("Signature: %d\n\n", t.Signature)
}

func (b *Block) PrintBlock() {
	fmt.Println("- - - - - - Printing Block - - - - - -")
	fmt.Printf("Index: %d\n", b.Index)
	fmt.Printf("Timestamp: %d\n", b.Timestamp)
	fmt.Printf("SlotLeader: %s\n", b.SlotLeader)
	fmt.Printf("Signature: %s\n", b.Signature)
	fmt.Printf("IndexHash: %x\n", b.IndexHash)
	fmt.Printf("MerkleRootHash: %x\n", b.MerkleRootHash)
	fmt.Println("StakeData:")
	for key, value := range b.StakeData {
		fmt.Printf("%s: %d,\t", key, value)
	}
	fmt.Println("Transactions:")
	for _, tx := range b.Transactions {
		tx.PrintTx()
	}
	fmt.Printf("PrevHash: %x\n", b.PrevHash)
	fmt.Printf("Hash: %x\n", b.Hash)
	fmt.Println("- - - - - - - - - - - - - - - - - - - -")
}
