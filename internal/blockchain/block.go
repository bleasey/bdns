package blockchain

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/rand"
	"crypto/sha256"
	"encoding/gob"
	"encoding/hex"
	"fmt"
	"log"
	"math/big"
	"sort"
	"time"
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
			stakeData[hex.EncodeToString(tx.OwnerKey)]++ // Increase stake for new domain registration
		} else if tx.Type == REVOKE {
			stakeData[hex.EncodeToString(tx.OwnerKey)]-- // Decrease stake for revoked domain
		}
	}

	return stakeData
}

func (b *Block) SignBlock(privateKey *ecdsa.PrivateKey) []byte {
	blockData := b.SerializeForSigning()
	hash := sha256.Sum256(blockData)

	r, s, err := ecdsa.Sign(rand.Reader, privateKey, hash[:])
	if err != nil {
		log.Panic("Failed to sign block:", err)
		return nil
	}

	// Ensure r and s are exactly 32 bytes each
	rBytes := r.Bytes()
	sBytes := s.Bytes()

	// ECDSA signatures are (r, s), each 32 bytes. Pad to ensure fixed size.
	rPadded := make([]byte, 32)
	sPadded := make([]byte, 32)
	copy(rPadded[32-len(rBytes):], rBytes)
	copy(sPadded[32-len(sBytes):], sBytes)

	// Concatenate r and s
	signature := append(rPadded, sPadded...)
	b.Signature = signature

	return signature
}

func (b *Block) VerifyBlock(publicKeyBytes []byte) bool {
	publicKey, err := BytesToPublicKey(publicKeyBytes)
	if err != nil {
		log.Println("Invalid public key format")
		return false
	}

	blockData := b.SerializeForSigning()
	hash := sha256.Sum256(blockData)

	if len(b.Signature) != 64 {
		log.Println("Invalid signature length")
		return false
	}

	r := new(big.Int).SetBytes(b.Signature[:32])
	s := new(big.Int).SetBytes(b.Signature[32:])

	return ecdsa.Verify(publicKey, hash[:], r, s)
}

func (b *Block) GetStakeDataBytes() []byte {
	// Extract keys from the map
	keys := make([]string, 0, len(b.StakeData))
	for key := range b.StakeData {
		keys = append(keys, key)
	}

	sort.Strings(keys) // Sort keys to ensure deterministic order

	stakeDataBytes := []byte{}
	for _, key := range keys {
		value := b.StakeData[key]
		decodedKey, _ := hex.DecodeString(key)
		stakeDataBytes = append(stakeDataBytes, decodedKey...)
		stakeDataBytes = append(stakeDataBytes, IntToByteArr(int64(value))...)
	}

	return stakeDataBytes
}

// Same as ComputeHash, but ommits Signature field
func (b *Block) SerializeForSigning() []byte {
	stakeDataBytes := b.GetStakeDataBytes()

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

func NewGenesisBlock(slotLeader []byte, privateKey *ecdsa.PrivateKey, registryKeys [][]byte, randomness []byte) *Block {
	stakeData := make(map[string]int)
	n := len(registryKeys)
	if n == 0 {
		log.Panic("No registries provided for genesis block")
	}

	// Initialize stakes to zero
	for _, key := range registryKeys {
		stakeData[hex.EncodeToString(key)] = 0
	}

	genesisBlock := Block{
		Index:          0,
		Timestamp:      time.Now().Unix(),
		SlotLeader:     slotLeader,
		Signature:      []byte{},
		IndexHash:      []byte{},
		MerkleRootHash: []byte{},
		StakeData:      stakeData,
		Transactions:   []Transaction{},
		PrevHash:       randomness, // Storing randomness in PrevHash field
		Hash:           []byte{},
	}

	genesisBlock.Signature = genesisBlock.SignBlock(privateKey)
	genesisBlock.Hash = genesisBlock.ComputeHash()

	return &genesisBlock
}

func (b *Block) ComputeHash() []byte {
	stakeDataBytes := b.GetStakeDataBytes()

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

func ValidateGenesisBlock(block *Block, registryKeys [][]byte, slotLeaderKey []byte) bool {
	if block.Index != 0 {
		return false
	}

	if !bytes.Equal(block.SlotLeader, slotLeaderKey) {
		return false
	}

	if !block.VerifyBlock(slotLeaderKey) {
		return false
	}

	if len(block.StakeData) != len(registryKeys) {
		return false
	}

	for _, key := range registryKeys {
		if block.StakeData[hex.EncodeToString(key)] != 0 {
			return false
		}
	}

	if len(block.Transactions) != 0 {
		return false
	}

	if !bytes.Equal(block.Hash, block.ComputeHash()) {
		fmt.Println("the false conition is block.Hash != block.ComputeHash()", !bytes.Equal(block.Hash, block.ComputeHash()))
		return false
	}

	return true
}

func ValidateBlock(newBlock *Block, oldBlock *Block, slotLeaderKey []byte) bool {
	if oldBlock.Index+1 != newBlock.Index {
		fmt.Println("the false conition is oldBlock.Index+1 != newBlock.Index", oldBlock.Index+1 != newBlock.Index)
		return false
	}

	if !bytes.Equal(oldBlock.Hash, newBlock.PrevHash) {
		return false
	}

	if !bytes.Equal(newBlock.SlotLeader, slotLeaderKey) {
		fmt.Println("the false conition is newBlock.SlotLeader != slotLeaderKey", !bytes.Equal(newBlock.SlotLeader, slotLeaderKey))
		return false
	}

	if !newBlock.VerifyBlock(slotLeaderKey) {
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

func (b *Block) ValidateTransactions() bool {
	for _, tx := range b.Transactions {
		if !VerifyTransaction(tx.OwnerKey, &tx) {
			return false
		}
	}
	return true
}
