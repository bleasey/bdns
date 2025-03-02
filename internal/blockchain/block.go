package blockchain

import (
	"bytes"
	"crypto/sha256"
	"crypto/ecdsa"
	"crypto/rand"
	"encoding/gob"
	"log"
	"time"
	"math/big"
)

type Block struct {
	Index          int64
	Timestamp      int64
	SlotLeader     []byte
	Signature      []byte
	IndexHash      []byte
	MerkleRootHash []byte
	StakeData      map[string]int // Registry Public Key -> Stake
	Transactions   []Transaction
	PrevHash       []byte
	Hash           []byte
}

func NewBlock(index int64, slotLeader []byte, indexHash []byte, transactions []Transaction, prevHash []byte, prevStakeData map[string]int, privateKey *ecdsa.PrivateKey) *Block {
	block := &Block{
		Index:        index,
		Timestamp:    time.Now().Unix(),
		SlotLeader:   slotLeader,
		IndexHash:    indexHash,
		Transactions: transactions,
		PrevHash:     prevHash,
	}

	block.MerkleRootHash = block.SetupMerkleTree()
	block.StakeData = block.SetupStakeData(prevStakeData)
	block.Signature = block.SignBlock(privateKey)
	block.Hash = block.ComputeHash()

	return block
}

func (b *Block) SetupStakeData(prevStakeData map[string]int) map[string]int {
	stakeData := make(map[string]int)
	// Copy previous stake data
	for key, value := range prevStakeData {
		stakeData[key] = value
	}

	// Update stake data based on transactions
	for _, tx := range b.Transactions {
		if tx.Type == REGISTER {
			stakeData[string(tx.OwnerKey)]++ // Increase stake for new domain registration
		} else if tx.Type == REVOKE {
			stakeData[string(tx.OwnerKey)]-- // Decrease stake for revoked domain
		}
	}

	return stakeData
}

func (b *Block) SignBlock(privateKey *ecdsa.PrivateKey) []byte {
	blockData := b.SerializeForSigning()
	hash := sha256.Sum256(blockData)

	r, s, err := ecdsa.Sign(rand.Reader, privateKey, hash[:])
	if err != nil {
		log.Panic(err)
		return nil
	}

	signature := append(r.Bytes(), s.Bytes()...)
	return signature
}

func (b *Block) VerifyBlock(publicKeyBytes []byte) bool {
	publicKey, err := BytesToPublicKey(publicKeyBytes)
	if err != nil {
		return false // Invalid public key format
	}

	blockData := b.SerializeForSigning()
	hash := sha256.Sum256(blockData)

	if len(b.Signature) < 64 {
		return false
	}

	r := new(big.Int).SetBytes(b.Signature[:32])
	s := new(big.Int).SetBytes(b.Signature[32:])

	return ecdsa.Verify(publicKey, hash[:], r, s)
}

// Same as ComputeHash, but ommits Signature field
func (b *Block) SerializeForSigning() []byte {
	stakeDataBytes := []byte{}
	for key, value := range b.StakeData {
		stakeDataBytes = append(stakeDataBytes, []byte(key)...)
		stakeDataBytes = append(stakeDataBytes, IntToByteArr(int64(value))...)
	}

	data := bytes.Join(
		[][]byte{
			IntToByteArr(b.Index),
			IntToByteArr(b.Timestamp),
			b.SlotLeader,
			b.IndexHash,
			b.MerkleRootHash,
			stakeDataBytes,
			b.PrevHash,
		},
		[]byte{},
	)

	hash := sha256.Sum256(data)
	return hash[:]
}

func NewGenesisBlock(registryKeys [][]byte, randomness []byte) Block {
	stakeData := make(map[string]int)
	n := len(registryKeys)
	if n == 0 {
		log.Panic("No registries provided for genesis block")
	}

	// Initialize stakes to zero
	for _, key := range registryKeys {
		stakeData[string(key)] = 0
	}

	genesisBlock := Block{
		Index:          0,
		Timestamp:      time.Now().Unix(),
		SlotLeader:     []byte{},
		Signature:      []byte{},
		IndexHash:      []byte{},
		MerkleRootHash: []byte{},
		StakeData:      stakeData,
		Transactions:   []Transaction{},
		PrevHash:       randomness, // Storing randomness in PrevHash field
		Hash:           []byte{},
	}

	genesisBlock.Hash = genesisBlock.ComputeHash()

	return genesisBlock
}

func (b *Block) ComputeHash() []byte {
	stakeDataBytes := []byte{}
	for key, value := range b.StakeData {
		stakeDataBytes = append(stakeDataBytes, []byte(key)...)
		stakeDataBytes = append(stakeDataBytes, IntToByteArr(int64(value))...)
	}

	data := bytes.Join(
		[][]byte{
			IntToByteArr(b.Index),
			IntToByteArr(b.Timestamp),
			b.SlotLeader,
			b.Signature,
			b.IndexHash,
			b.MerkleRootHash,
			stakeDataBytes,
			b.PrevHash,
		},
		[]byte{},
	)

	hash := sha256.Sum256(data)
	return hash[:]
}

// Creates a merkle tree from the block's transactions and returns the root hash
func (b *Block) SetupMerkleTree() []byte {
	var transactions [][]byte

	for _, tx := range b.Transactions {
		transactions = append(transactions, tx.Serialize())
	}
	mTree := NewMerkleTree(transactions)

	return mTree.RootNode.Data
}

func (b *Block) Serialize() []byte {
	var result bytes.Buffer
	encoder := gob.NewEncoder(&result)

	err := encoder.Encode(b)
	if err != nil {
		log.Panic(err)
	}

	return result.Bytes()
}

func DeserializeBlock(d []byte) *Block {
	var block Block

	decoder := gob.NewDecoder(bytes.NewReader(d))
	err := decoder.Decode(&block)
	if err != nil {
		log.Panic(err)
	}

	return &block
}

func (b *Block) ValidateBlock(newBlock Block, oldBlock Block, slotLeader []byte, leaderPubKey []byte) bool {
	if oldBlock.Index+1 != newBlock.Index {
		return false
	}

	if !bytes.Equal(oldBlock.Hash, newBlock.PrevHash) {
		return false
	}

	if !bytes.Equal(newBlock.SlotLeader, slotLeader) {
		return false
	}
	
	if !newBlock.VerifyBlock(leaderPubKey) {
		return false
	}

	if !bytes.Equal(newBlock.MerkleRootHash, newBlock.SetupMerkleTree()) {
		return false
	}

	if !areStakesEqual(newBlock.StakeData, newBlock.SetupStakeData(oldBlock.StakeData)) {
		return false
	}

	if !newBlock.ValidateTransactions() {
		return false
	}

	if !bytes.Equal(newBlock.Hash, newBlock.ComputeHash()) {
		return false
	}

	return true
}

func (b *Block) ValidateTransactions () bool {
	for _, tx := range b.Transactions {
		if !VerifyTransaction(tx.OwnerKey, &tx) {
			return false
		}
	}
	return true
}
