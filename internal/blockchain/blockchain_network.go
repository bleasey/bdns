package blockchain

import (
	"encoding/gob"
	"log"
	"net"
	"bytes"
	"sync"

	"github.com/boltdb/bolt"
)

// SendBlockchain sends the entire blockchain to a peer over a network connection
func (bc *Blockchain) SendBlockchain(conn net.Conn) {
	defer conn.Close()

	encoder := gob.NewEncoder(conn)
	iter := bc.Iterator()

	for {
		block := iter.Next()
		if block == nil {
			break
		}

		err := encoder.Encode(block)
		if err != nil {
			log.Println("Error sending block:", err)
			return
		}
	}
}

// Replaces the current blockchain with a new valid one received over the connection.
func (bc *Blockchain) ReplaceChain(conn net.Conn, mutex *sync.Mutex) {
	defer conn.Close()

	decoder := gob.NewDecoder(conn)
	var newBlocks []*Block

	// Decode all received blocks
	for {
		var block Block
		err := decoder.Decode(&block)
		if err != nil {
			break // Stop if there are no more blocks
		}
		newBlocks = append(newBlocks, &block)
	}

	// Validate new blockchain before replacing
	if len(newBlocks) == 0 {
		log.Println("Received blockchain is empty, ignoring replacement.")
		return
	}

	mutex.Lock()
	defer mutex.Unlock()

	// Check if the received blockchain is longer and valid
	if len(newBlocks) > bc.GetLength() && bc.IsValidChain(newBlocks) {
		bc.replace(newBlocks)
		log.Println("Blockchain replaced successfully.")
	} else {
		log.Println("Received blockchain is invalid or not longer.")
	}
}

// Validates the integrity of the given blockchain.
func (bc *Blockchain) IsValidChain(newBlocks []*Block) bool {
	for i := 1; i < len(newBlocks); i++ {
		if !bytes.Equal(newBlocks[i].PrevHash, newBlocks[i-1].Hash) {
			return false
		}
	}
	return true
}

// Replace swaps the blockchain with the new valid one.
func (bc *Blockchain) replace(newBlocks []*Block) {
	db := bc.db
	db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte("blocks"))
		if b == nil {
			return nil
		}

		// Remove old blockchain
		err := tx.DeleteBucket([]byte("blocks"))
		if err != nil {
			return err
		}

		// Create new bucket and store new blocks
		b, err = tx.CreateBucket([]byte("blocks"))
		if err != nil {
			return err
		}

		for _, block := range newBlocks {
			err := b.Put(block.Hash, block.Serialize())
			if err != nil {
				return err
			}
		}

		// Update the tip
		err = tx.Bucket([]byte("blocks")).Put([]byte("l"), newBlocks[len(newBlocks)-1].Hash)
		if err != nil {
			return err
		}
		bc.tip = newBlocks[len(newBlocks)-1].Hash
		return nil
	})
}

// Returns the length of the current blockchain.
func (bc *Blockchain) GetLength() int {
	iter := bc.Iterator()
	count := 0
	for iter.Next() != nil {
		count++
	}
	return count
}
