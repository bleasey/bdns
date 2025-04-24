package network

import (
	"bytes"
	"encoding/hex"
	"log"

	"github.com/bleasey/bdns/internal/blockchain"
)

func (n *Node) AddBlock(block *blockchain.Block) {
	epoch := (block.Timestamp - n.Config.InitialTimestamp) / (n.Config.SlotInterval * n.Config.SlotsPerEpoch)
	slotLeader := n.GetSlotLeader(epoch)

	// Verify received block
	if (block.Index == 0 && !blockchain.ValidateGenesisBlock(block, n.RegistryKeys, slotLeader)) ||
		(block.Index != 0 && !blockchain.ValidateBlock(block, n.Blockchain.GetLatestBlock(), slotLeader)) {
		log.Println("Invalid block received at ", n.Address)
		return
	}

	// Update index tree
	n.TxMutex.Lock()
	activationEpoch := epoch + n.Config.RegistryUpdateAfter
	for _, tx := range block.Transactions {
		switch tx.Type {
		case blockchain.REGISTER:
			n.IndexManager.Add(tx.DomainName, &tx)

		case blockchain.UPDATE:
			n.IndexManager.Update(tx.DomainName, &tx)

		case blockchain.REVOKE:
			n.IndexManager.Remove(tx.DomainName)

		default:
			// For registry operations

			// REGISTRY_UPDATE & REGISTRY_SELF_REVOKE should be from the owner
			if (tx.Type == blockchain.REGISTRY_UPDATE || tx.Type == blockchain.REGISTRY_SELF_REVOKE) &&
				!bytes.Equal(tx.TargetKey, tx.OwnerKey) {
				log.Printf("Invalid registry operation from %s at %s\n", n.Address, tx.TargetKey)
				continue
			}

			if (tx.Type == blockchain.REGISTRY_FORCE_REVOKE) {
				// Check if the sender is a slot leader
				if !bytes.Equal(tx.TargetKey, slotLeader) {
					log.Printf("Invalid registry operation from %s at %s\n", n.Address, tx.TargetKey)
					continue
				}

				// Check if revocation is valid
				n.VoteMutex.Lock()
				voteEpoch := (epoch / n.Config.EpochsPerVote) - 1 // Vote in prev VoteEpoch
				voteCount := n.VoteCount[voteEpoch][hex.EncodeToString(tx.TargetKey)]
				n.VoteMutex.Unlock()
				
				voteCuttoff := int64(n.Config.VoteRevokeCuttoff * float32(len(n.RegistryKeys)))

				if voteCount < voteCuttoff {
					log.Printf("Invalid registry operation from %s at %s\n", n.Address, tx.TargetKey)
					continue
				}
			}

			n.RegistryOps[activationEpoch] = append(n.RegistryOps[activationEpoch], RegistryOp{
				OP: tx.Type,
				Key: tx.TargetKey,
				UpdatedKey: tx.UpdatedKey,
			})
		}
	}
	blockchain.RemoveTxsFromPool(block.Transactions, n.TransactionPool)
	n.TxMutex.Unlock()

	// Add block to blockchain
	n.BcMutex.Lock()
	defer n.BcMutex.Unlock()
	n.Blockchain.AddBlock(block)
}

func (n *Node) AddTransaction(tx *blockchain.Transaction) {
	n.TxMutex.Lock()
	defer n.TxMutex.Unlock()
	n.TransactionPool[tx.TID] = tx
}

func (n *Node) UpdateRegistries(epoch int64) {
	n.RegistryMutex.Lock()
	defer n.RegistryMutex.Unlock()

	registryOps, exists := n.RegistryOps[epoch]
	if !exists {
		return
	}

	for _, op := range registryOps {
		switch op.OP {
		case blockchain.REGISTRY_REGISTER:
			n.RegistryKeys = append(n.RegistryKeys, op.Key)

		case blockchain.REGISTRY_UPDATE:
			for i, key := range n.RegistryKeys {
				if bytes.Equal(op.Key, key) {
					n.RegistryKeys[i] = op.UpdatedKey
					break
				}
			}

		default:
			// REGISTRY_SELF_REVOKE, REGISTRY_FORCE_REVOKE
			for i, key := range n.RegistryKeys {
				if bytes.Equal(op.Key, key) {
					n.RegistryKeys = append(n.RegistryKeys[:i], n.RegistryKeys[i+1:]...)
					break
				}
			}
		}
	}
}

// HandleINV processes inventory message and requests missing blocks
func (n *Node) HandleINV(sender string) {
	n.BcMutex.Lock()
	defer n.BcMutex.Unlock()

	localHeight := n.Blockchain.GetLatestBlock().Index

	getBlockMsg := map[string]int{
		"height": int(localHeight),
	}
	n.P2PNetwork.DirectMessage(MsgGetBlock, getBlockMsg, sender)
	log.Printf("[INV] %s requested blocks from height %d\n", n.Address, localHeight)
}

// HandleGetData responds with transactions in the mempool
func (n *Node) HandleGetData(sender string) {
	n.TxMutex.Lock()
	defer n.TxMutex.Unlock()

	for _, tx := range n.TransactionPool {
		n.P2PNetwork.DirectMessage(MsgTransaction, tx, sender)
	}
	log.Printf("[GETDATA] %s sent mempool transactions to %s\n", n.Address, sender)
}

// HandleGetBlock responds with full blockchain blocks starting from a given height
func (n *Node) HandleGetBlock(sender string) {
	n.BcMutex.Lock()
	defer n.BcMutex.Unlock()

	// Respond with blocks newer than peer's height
	start := n.Blockchain.GetLatestBlock().Index - 5 // sending last 5 blocks for now
	if start < 0 {
		start = 0
	}

	blocks := n.Blockchain.GetBlocksFrom(int(start))
	for _, b := range blocks {
		n.P2PNetwork.DirectMessage(MsgBlock, b, sender)
	}
	log.Printf("[GETBLOCK] %s sent recent blocks to %s\n", n.Address, sender)
}

// HandleMerkleRequest sends Merkle proof path for a record
func (n *Node) HandleMerkleRequest(sender string) {
	n.BcMutex.Lock()
	defer n.BcMutex.Unlock()

	// Simplified: just sending full block instead of real Merkle path
	// Ideally: compute Merkle root and proof path
	latest := n.Blockchain.GetLatestBlock()
	if latest != nil {
		n.P2PNetwork.DirectMessage(MsgBlock, latest, sender)
		log.Printf("[MERKLE] %s sent block with Merkle data to %s\n", n.Address, sender)
	}
}
