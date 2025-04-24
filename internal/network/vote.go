package network

import (
	"encoding/hex"
	"log"
	"crypto/ecdsa"
	"crypto/rand"
	"crypto/sha256"
	"math/big"

	"github.com/bleasey/bdns/internal/blockchain"
)

type RegistryVote struct {
	Timestamp	int64
	VoterKey	[]byte
	Vote		[]byte
	Signature	[]byte
}

func (n *Node) CheckVotes(epoch int64, transactions []blockchain.Transaction) []blockchain.Transaction {
	n.VoteMutex.Lock()
	defer n.VoteMutex.Unlock()
	n.RegistryMutex.Lock()
	defer n.RegistryMutex.Unlock()

	voteCuttoff := int64(n.Config.VoteRevokeCuttoff * float32(len(n.RegistryKeys)))
	voteEpoch := (epoch / n.Config.EpochsPerVote) - 1 // Vote in prev VoteEpoch
	for registry, count := range n.VoteCount[voteEpoch] {
		if count >= voteCuttoff {
			tx := blockchain.NewRegistryTransaction(blockchain.REGISTRY_FORCE_REVOKE, []byte(registry), nil, n.KeyPair.PublicKey, &n.KeyPair.PrivateKey, n.TransactionPool)
			transactions = append(transactions, *tx)
		}
	}

	return transactions
}

func (n *Node) HandleVoteKick(voteMsg RegistryVote) {
	n.VoteMutex.Lock()
	defer n.VoteMutex.Unlock()

	if !VerifyVote(voteMsg.VoterKey, &voteMsg) {
		log.Printf("[VOTE] %s invalid vote signature\n", n.Address)
		return
	}

	voteEpoch := (voteMsg.Timestamp - n.Config.InitialTimestamp) / (n.Config.SlotInterval * n.Config.SlotsPerEpoch * n.Config.EpochsPerVote)
	_, voteExists := n.HasVoted[voteEpoch][hex.EncodeToString(voteMsg.VoterKey)][hex.EncodeToString(voteMsg.Vote)]
	if voteExists {
		log.Printf("[VOTE] %s already voted in this epoch\n", n.Address)
		return
	}

	n.VoteCount[voteEpoch][hex.EncodeToString(voteMsg.Vote)]++
	log.Printf("[VOTE] %s voted to kick %s\n", n.Address, voteMsg.Vote)
}

func SignVote(privateKey *ecdsa.PrivateKey, vote *RegistryVote) []byte {
	voteData := vote.SerializeForSigning()
	hash := sha256.Sum256(voteData)

	r, s, err := ecdsa.Sign(rand.Reader, privateKey, hash[:])
	if err != nil {
		log.Panic("Failed to sign vote:", err)
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
	vote.Signature = signature

	return signature
}

func VerifyVote(publicKeyBytes []byte, vote *RegistryVote) bool {
	publicKey, err := blockchain.BytesToPublicKey(publicKeyBytes)
	if err != nil {
		log.Println("Invalid vote public key format")
		return false
	}

	voteData := vote.SerializeForSigning()
	hash := sha256.Sum256(voteData)

	// Ensure signature length is correct
	if len(vote.Signature) != 64 {
		log.Println("Invalid vote signature length")
		return false
	}

	// Extract r and s from the signature
	r := new(big.Int).SetBytes(vote.Signature[:32])
	s := new(big.Int).SetBytes(vote.Signature[32:])

	return ecdsa.Verify(publicKey, hash[:], r, s)
}

func (vote *RegistryVote) SerializeForSigning() []byte {
	voteData := append(blockchain.IntToByteArr(int64(vote.Timestamp)), vote.VoterKey...)
	voteData = append(voteData, vote.Vote...)

	return voteData
}
