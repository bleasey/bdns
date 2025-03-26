package network

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"time"

	"github.com/bleasey/bdns/internal/blockchain"
	"github.com/bleasey/bdns/internal/consensus"
)

func (n *Node) AddBlock(block *blockchain.Block) {
	epoch := (block.Timestamp - n.Config.InitialTimestamp) / n.Config.EpochInterval
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
		slotLeader = consensus.GetSlotLeader(int(epoch), int(n.Config.Seed), n.RegistryKeys, nil)
	} else {
		n.BcMutex.Lock()
		latestBlock := n.Blockchain.GetLatestBlock()
		n.BcMutex.Unlock()
		slotLeader = consensus.GetSlotLeader(int(epoch), int(n.Config.Seed), n.RegistryKeys, latestBlock.StakeData)
	}

	n.SlotLeaders[epoch] = slotLeader
	return slotLeader
}

func (n *Node) DNSRequestHandler(req BDNSRequest, conn net.Conn) {
	n.TxMutex.Lock()
	defer n.TxMutex.Unlock()
	fmt.Println("DNS Request received from:", conn.RemoteAddr(), " at:", n.Address, " Domain Name:", req.DomainName)

	tx := n.IndexManager.GetIP(req.DomainName)
	if tx != nil {
		res := BDNSResponse{
			Timestamp:  tx.Timestamp,
			DomainName: tx.DomainName,
			IP:         tx.IP,
			TTL:        tx.TTL,
			OwnerKey:   tx.OwnerKey,
			Signature:  tx.Signature,
		}
		data, _ := json.Marshal(res)
		msg := Message{Sender: n.Address, Type: DNSResponse, Data: data}
		// n.SendToPeer(conn, msg)
		n.Broadcast(msg)
	}else {
		// If BDNS lookup fails, try resolving via traditional DNS
		ip, err := ResolveLegacyDNS(req.DomainName)
		if err == nil {
			fmt.Printf("Node %s resolved %s via legacy DNS: %s\n", n.Address, req.DomainName, ip)
			n.SendToPeer(conn, Message{Type: DNSResponse, Data: []byte(ip)})
		} else {
			fmt.Printf("Node %s could not resolve %s\n", n.Address, req.DomainName)
		}
	}

	}

func (n *Node) DNSResponseHandler(res BDNSResponse, conn net.Conn) {
	fmt.Println("DNS Response received from:", conn.RemoteAddr(), " at:", n.Address, " Domain Name:", res.DomainName, " IP:", res.IP)
}

func (n *Node) MakeDNSRequest(domainName string) {
	req := BDNSRequest{DomainName: domainName}
	data, _ := json.Marshal(req)
	msg := Message{Sender: n.Address, Type: DNSRequest, Data: data}
	n.Broadcast(msg)
}

func (n *Node) BroadcastTransaction(tx blockchain.Transaction) {
	n.AddTransaction(&tx) // Add tx to own map
	data, _ := json.Marshal(tx)
	msg := Message{Sender: n.Address, Type: MsgTransaction, Data: data}
	n.Broadcast(msg)
}

func (n *Node) BroadcastBlock(block blockchain.Block) {
	data, _ := json.Marshal(block)
	msg := Message{Sender: n.Address, Type: MsgBlock, Data: data}
	n.Broadcast(msg)
}

func (n *Node) CreateBlockIfLeader(epochInterval int64) {
	ticker := time.NewTicker(time.Duration(epochInterval) * time.Second)
	defer ticker.Stop()
	epoch := int64(-1) // Initialize epoch counter

	for range ticker.C {
		epoch++
		if epoch == 0 {
			currSlotLeader := n.GetSlotLeader(epoch)

			if !bytes.Equal(currSlotLeader, n.KeyPair.PublicKey) {
				continue
			}

			// Create genesis block
			fmt.Println("Node", n.Address, "is the slot leader for the genesis block")

			seedBytes := []byte(fmt.Sprintf("%d", n.Config.Seed))
			genesisBlock := blockchain.NewGenesisBlock(currSlotLeader, &n.KeyPair.PrivateKey, n.RegistryKeys, seedBytes)

			n.BcMutex.Lock()
			n.Blockchain.AddBlock(genesisBlock)
			n.BcMutex.Unlock()

			n.BroadcastBlock(*genesisBlock)
			fmt.Print("Genesis block created and broadcasted by node ", n.Address, "\n\n")
		} else {
			currSlotLeader := n.GetSlotLeader(epoch)

			if !bytes.Equal(currSlotLeader, n.KeyPair.PublicKey) {
				continue
			}

			// Create block from transactions
			fmt.Printf("Node %s is the slot leader, creating a block...\n", n.Address)

			// Get transactions from the pool
			n.TxMutex.Lock()
			var transactions []blockchain.Transaction
			for _, tx := range n.TransactionPool {
				transactions = append(transactions, *tx)
			}
			n.TransactionPool = make(map[int]*blockchain.Transaction) // Clear pool
			n.TxMutex.Unlock()

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

			n.BroadcastBlock(*newBlock)
			fmt.Print("Block ", newBlock.Index, " created and broadcasted by node ", n.Address, "\n\n")
		}
	}
}
