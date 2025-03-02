package blockchain

import (
	"errors"
	"fmt"
	"github.com/boltdb/bolt"
	"log"
	"os"
)

const dbFile = "chaindata/blockchain_%s.db"
const blocksBucket = "blocks"

// Blockchain implements interactions with a DB
type Blockchain struct {
	tip []byte
	db  *bolt.DB
}

// CreateBlockchain creates a new blockchain DB
func CreateBlockchain(chainID string, registryKeys [][]byte, randomness []byte) *Blockchain {
	dbFile := fmt.Sprintf(dbFile, chainID)
	if dbExists(dbFile) {
		fmt.Println("Blockchain already exists.")
		os.Exit(1)
	}

	var tip []byte

	genesis := NewGenesisBlock(registryKeys, randomness)

	db, err := bolt.Open(dbFile, 0600, nil)
	if err != nil {
		log.Panic(err)
	}

	err = db.Update(func(tx *bolt.Tx) error {
		b, err := tx.CreateBucket([]byte(blocksBucket))
		if err != nil {
			log.Panic(err)
		}

		// Add a Hash -> Block mapping for the genesis block
		err = b.Put(genesis.Hash, genesis.Serialize())
		if err != nil {
			log.Panic(err)
		}

		// Add a key for the last block hash
		err = b.Put([]byte("l"), genesis.Hash)
		if err != nil {
			log.Panic(err)
		}
		tip = genesis.Hash

		return nil
	})
	if err != nil {
		log.Panic(err)
	}

	bc := Blockchain{tip, db}

	return &bc
}

// OpenBlockchain opens an existing blockchain DB
func OpenBlockchain(chainID string) *Blockchain {
	dbFile := fmt.Sprintf(dbFile, chainID)
	if !dbExists(dbFile) {
		fmt.Println("No existing blockchain found. Create one first.")
		os.Exit(1)
	}

	var tip []byte
	db, err := bolt.Open(dbFile, 0600, nil)
	if err != nil {
		log.Panic(err)
	}

	err = db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(blocksBucket))
		tip = b.Get([]byte("l"))

		return nil
	})
	if err != nil {
		log.Panic(err)
	}

	bc := Blockchain{tip, db}

	return &bc
}

// AddBlock saves the block into the blockchain
func (bc *Blockchain) AddBlock(block *Block) {
	err := bc.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(blocksBucket))
		blockInDB := b.Get(block.Hash)

		if blockInDB != nil {
			return nil
		}

		blockData := block.Serialize()
		err := b.Put(block.Hash, blockData)
		if err != nil {
			log.Panic(err)
		}

		lastHash := b.Get([]byte("l"))
		lastBlockData := b.Get(lastHash)
		lastBlock := DeserializeBlock(lastBlockData)

		// Add block only if it belongs to a longer chain
		if block.Index > lastBlock.Index {
			err = b.Put([]byte("l"), block.Hash)
			if err != nil {
				log.Panic(err)
			}
			bc.tip = block.Hash
		}

		return nil
	})
	if err != nil {
		log.Panic(err)
	}
}

// FindTransaction finds a transaction by its ID
func (bc *Blockchain) FindTransaction(tid int) (Transaction, error) {
	bci := bc.Iterator()

	for {
		block := bci.Next()

		for _, tx := range block.Transactions {
			if tx.TID == tid {
				return tx, nil
			}
		}

		if block.Index == 0 {
			break
		}
	}

	return Transaction{}, errors.New("Transaction is not found")
}

// Iterator returns a BlockchainIterat
func (bc *Blockchain) Iterator() *BlockchainIterator {
	bci := &BlockchainIterator{bc.tip, bc.db}

	return bci
}

// GetBestHeight returns the height (Index) of the latest block
func (bc *Blockchain) GetBestHeight() int {
	var lastBlock Block

	err := bc.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(blocksBucket))
		lastHash := b.Get([]byte("l"))
		blockData := b.Get(lastHash)
		lastBlock = *DeserializeBlock(blockData)

		return nil
	})
	if err != nil {
		log.Panic(err)
	}

	return int(lastBlock.Index)
}

// GetBlock finds a block by its hash and returns it
func (bc *Blockchain) GetBlock(blockHash []byte) (Block, error) {
	var block Block

	err := bc.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(blocksBucket))

		blockData := b.Get(blockHash)

		if blockData == nil {
			return errors.New("Block is not found")
		}

		block = *DeserializeBlock(blockData)

		return nil
	})
	if err != nil {
		return block, err
	}

	return block, nil
}

// GetBlockHashes returns a list of hashes of all the blocks in the chain
func (bc *Blockchain) GetBlockHashes() [][]byte {
	var blocks [][]byte
	bci := bc.Iterator()

	for {
		block := bci.Next()

		blocks = append(blocks, block.Hash)

		if block.Index == 0 {
			break
		}
	}

	return blocks
}

func (bc *Blockchain) GetLatestBlock() *Block {
	var block *Block

	err := bc.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(blocksBucket))
		lastHash := b.Get([]byte("l"))
		blockData := b.Get(lastHash)
		block = DeserializeBlock(blockData)

		return nil
	})
	if err != nil {
		log.Panic(err)
	}

	return block
}
