package blockchain

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/elliptic"
	"encoding/binary"
	"errors"
	"log"
	"math/big"
	"os"
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

func BytesToPublicKey(publicKeyBytes []byte) (*ecdsa.PublicKey, error) {
	if len(publicKeyBytes) != 64 {
		return nil, errors.New("invalid public key length")
	}

	curve := elliptic.P256()
	x := new(big.Int).SetBytes(publicKeyBytes[:32])
	y := new(big.Int).SetBytes(publicKeyBytes[32:])

	return &ecdsa.PublicKey{Curve: curve, X: x, Y: y}, nil
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
