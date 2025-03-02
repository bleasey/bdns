package network

import (
	"encoding/json"
	"fmt"
	"log"
	"net"
	"time"
	"bytes"

	"github.com/bleasey/bdns/internal/blockchain"
	"github.com/bleasey/bdns/internal/consensus"
)

func (n *Node) AddBlock(block *blockchain.Block) {
	// Update index tree
	n.TxMutex.Lock()
	defer n.TxMutex.Unlock()
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

func (n *Node) DNSRequestHandler(req BDNSRequest, conn net.Conn) {
	n.TxMutex.Lock()
	defer n.TxMutex.Unlock()
	tx := n.IndexManager.GetIP(req.DomainName)
	if tx != nil {
		res := BDNSResponse{
			Timestamp: tx.Timestamp,
			DomainName: tx.DomainName,
			IP: tx.IP,
			TTL: tx.TTL,
			OwnerKey: tx.OwnerKey,
			Signature: tx.Signature,
		}
		data, _ := json.Marshal(res)
		msg := Message{Type: DNSResponse, Data: data}
		n.SendToPeer(conn, msg)
	}
	log.Println("DNS Request from:", conn.RemoteAddr(), " To:", n.Address, " Domain Name:", req.DomainName)
}

func (n *Node) DNSResponseHandler(res BDNSResponse, conn net.Conn) {
	log.Println("DNS Response from:", conn.RemoteAddr(), " To:", n.Address, " Domain Name:", res.DomainName, " IP:", res.IP)
}

func (n *Node) MakeDNSRequest(domainName string) {
	req := BDNSRequest{DomainName: domainName}
	data, _ := json.Marshal(req)
	msg := Message{Type: DNSRequest, Data: data}
	n.Broadcast(msg)
}

func (n *Node) BroadcastTransaction(tx blockchain.Transaction) {
    data, _ := json.Marshal(tx)
    msg := Message{Type: MsgTransaction, Data: data}
    n.Broadcast(msg)
}

func (n *Node) BroadcastBlock(block blockchain.Block) {
    data, _ := json.Marshal(block)
    msg := Message{Type: MsgBlock, Data: data}
    n.Broadcast(msg)
}

func (n *Node) CreateBlockIfLeader(epochInterval int, seed int) {
    ticker := time.NewTicker(time.Duration(epochInterval) * time.Second)
    defer ticker.Stop()
	epoch := 0 // Initialize epoch counter

    for range ticker.C {
		n.BcMutex.Lock()
		latestBlock := n.Blockchain.GetLatestBlock()
		n.BcMutex.Unlock()

		leader := consensus.GetSlotLeader(epoch, seed, n.RegistryKeys, latestBlock.StakeData)

        if bytes.Equal(leader, n.KeyPair.PublicKey) {
            fmt.Printf("Node %s is the slot leader, creating a block...\n", n.Address)

            n.BcMutex.Lock()

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
                n.BcMutex.Unlock()
                continue
            }

			newBlock := blockchain.NewBlock(int64(epoch), n.KeyPair.PublicKey, nil, transactions, latestBlock.Hash, latestBlock.StakeData, &n.KeyPair.PrivateKey)
            n.Blockchain.AddBlock(newBlock)
            n.BcMutex.Unlock()
            n.BroadcastBlock(*newBlock)
        }

		epoch++ 
    }
}
