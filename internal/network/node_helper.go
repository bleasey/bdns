package network

import (
	"bytes"
	"fmt"
	"log"
	"time"

	"github.com/bleasey/bdns/internal/blockchain"
	"github.com/bleasey/bdns/internal/consensus"
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
	for _, tx := range block.Transactions {
		switch tx.Type {
		case blockchain.REGISTER:
			n.IndexManager.Add(tx.DomainName, &tx)

		case blockchain.UPDATE:
			n.IndexManager.Update(tx.DomainName, &tx)

		case blockchain.REVOKE:
			n.IndexManager.Remove(tx.DomainName)
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

func (n *Node) GetSlotLeader(epoch int64) []byte {
	n.SlotMutex.Lock()
	defer n.SlotMutex.Unlock()

	slotLeader, exists := n.SlotLeaders[epoch]
	if exists {
		return slotLeader
	}

	// Assuming map miss only happens for current epoch
	if epoch == 0 {
		slotLeader = consensus.GetSlotLeaderUtil(n.RegistryKeys, nil, n.EpochRandoms[epoch])
	} else {
		n.BcMutex.Lock()
		latestBlock := n.Blockchain.GetLatestBlock()
		n.BcMutex.Unlock()

		slotLeader = consensus.GetSlotLeaderUtil(n.RegistryKeys, latestBlock.StakeData, n.EpochRandoms[epoch])
	}

	n.SlotLeaders[epoch] = slotLeader
	return slotLeader
}

func (n *Node) CreateBlockIfLeader() {
	ticker := time.NewTicker(time.Duration(n.Config.SlotInterval) * time.Second)
	defer ticker.Stop()

	// GENESIS BLOCK creation
	currSlotLeader := n.RegistryKeys[0] // Default genesis slot leader

	if bytes.Equal(currSlotLeader, n.KeyPair.PublicKey) {
		fmt.Println("Node", n.Address, "is the slot leader for the genesis block")

		seedBytes := []byte(fmt.Sprintf("%f", n.Config.Seed))
		genesisBlock := blockchain.NewGenesisBlock(currSlotLeader, &n.KeyPair.PrivateKey, n.RegistryKeys, seedBytes)

		n.BcMutex.Lock()
		n.Blockchain.AddBlock(genesisBlock)
		n.BcMutex.Unlock()

		n.P2PNetwork.BroadcastMessage(MsgBlock, *genesisBlock)
		fmt.Print("Genesis block created and broadcasted by node ", n.Address, "\n\n")
	}
	n.BroadcastRandomNumber(1, n.RegistryKeys) // Broadcast nums for the fist epoch
	time.Sleep(time.Duration(n.Config.SlotInterval * n.Config.SlotsPerEpoch) * time.Second) // wait till end of epoch

	// Initialize loop variables
	slot := int64(n.Config.SlotsPerEpoch - 1)
	epoch := int64(0)
	blockTxLimit := 10

	// Ticker loop for slots
	for range ticker.C {
		slot++
		newEpoch := slot / n.Config.SlotsPerEpoch
		
		// Update epoch and leader only when the epoch changes
		if newEpoch != epoch {
			epoch = newEpoch
			currSlotLeader = n.GetSlotLeader(epoch)
			n.BroadcastRandomNumber(epoch+1, n.RegistryKeys) // Send rand nums for next epoch
		}

		// Only the current slot leader should produce a block
		if !bytes.Equal(currSlotLeader, n.KeyPair.PublicKey) {
			continue
		}

		// Create block from transactions
		fmt.Printf("Node %s is the slot leader, creating a block...\n", n.Address)

		transactions := n.ChooseTxFromPool(blockTxLimit)

		if len(transactions) == 0 {
			fmt.Println("No transactions to add. Skipping block creation.")
			continue
		}

		fmt.Println("Transactions in block:", len(transactions))

		n.BcMutex.Lock()
		latestBlock := n.Blockchain.GetLatestBlock()
		newBlock := blockchain.NewBlock(latestBlock.Index+1, currSlotLeader, nil, transactions, latestBlock.Hash, latestBlock.StakeData, &n.KeyPair.PrivateKey)
		n.Blockchain.AddBlock(newBlock)
		n.BcMutex.Unlock()

		n.P2PNetwork.BroadcastMessage(MsgBlock, *newBlock)
		fmt.Print("Block ", newBlock.Index, " created and broadcasted by node ", n.Address, "\n\n")
	}
}

func (n *Node) ChooseTxFromPool(limit int) []blockchain.Transaction {
	n.TxMutex.Lock()
	defer n.TxMutex.Unlock()

	if len(n.TransactionPool) == 0 {
		return nil
	}

	transactions := make([]blockchain.Transaction, 0, limit)
	for _, tx := range n.TransactionPool {
		if len(transactions) >= limit {
			break
		}
		transactions = append(transactions, *tx)
		delete(n.TransactionPool, tx.TID)
	}

	return transactions
}
