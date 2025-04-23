package network

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"sync"

	"github.com/bleasey/bdns/internal/blockchain"
	"github.com/bleasey/bdns/internal/consensus"
	"github.com/bleasey/bdns/internal/index"
	"github.com/libp2p/go-libp2p/core/network"
)

// Node represents a blockchain peer
type Node struct {
	Address         string
	Port            int
	Config          NodeConfig
	P2PNetwork      *P2PNetwork
	KeyPair         *blockchain.KeyPair
	RegistryKeys    [][]byte
	RegistryOps		map[int64][]RegistryOp
	RegistryMutex   sync.Mutex
	SlotLeaders     map[int64][]byte // epoch to slot leader
	SlotMutex       sync.Mutex
	TransactionPool map[int]*blockchain.Transaction
	TxMutex         sync.Mutex
	IndexManager    *index.IndexManager
	Blockchain      *blockchain.Blockchain
	BcMutex         sync.Mutex
	RandomNumber    []byte
	RandomMutex     sync.Mutex
	EpochRandoms    map[int64]map[string]consensus.SecretValues
	IsFullNode      bool // full vs light node
	PeerID          string
	KnownFullPeers  []string
}

// Node Config
type NodeConfig struct {
	InitialTimestamp 	int64
	SlotInterval     	int64
	SlotsPerEpoch    	int64
	Seed             	float64
	RegistryUpdateAfter int64
}

type RandomNumberMsg struct {
	Epoch       int64
	SecretValue int    // u_i value
	RandomValue int    // r_i value
	Sender      []byte // Registry's public key
}

type RegistryOp struct {
	OP 			blockchain.TransactionType
	Key     	[]byte
	UpdatedKey	[]byte // in case of updation
}

// NewNode initializes a blockchain node
func NewNode(ctx context.Context, addr string, topicName string, isFullNode bool) (*Node, error) {
	p2p, err := NewP2PNetwork(ctx, addr, topicName)
	if err != nil {
		return nil, err
	}

	node := &Node{
		Address:         p2p.Host.Addrs()[0].String(),
		P2PNetwork:      p2p,
		KeyPair:      	 blockchain.NewKeyPair(),
		RegistryOps: 	 make(map[int64][]RegistryOp),
		SlotLeaders:     make(map[int64][]byte),
		TransactionPool: make(map[int]*blockchain.Transaction),
		IndexManager:    index.NewIndexManager(),
		Blockchain:      nil,
		EpochRandoms:    make(map[int64]map[string]consensus.SecretValues),
		IsFullNode:      isFullNode,
		PeerID:          p2p.Host.ID().String(),
		KnownFullPeers:  []string{},
	}

	go node.ListenForDirectMessages()
	go node.P2PNetwork.ListenForGossip()
	go node.HandleGossip()
	return node, nil
}

func (n *Node) GenerateRandomNumber() []byte {
	n.RandomMutex.Lock()
	defer n.RandomMutex.Unlock()

	randomBytes := make([]byte, 32)
	if _, err := rand.Read(randomBytes); err != nil {
		log.Panic("Failed to generate random number:", err)
	}

	n.RandomNumber = randomBytes
	return randomBytes
}

func (n *Node) HandleMsgGivenType(msg GossipMessage) {
	// This includes messages received from BOTH direct and broadcast mode
	// since for now there is no difference in handling based on the mode of reception
	switch msg.Type {
	case DNSRequest:
		var req BDNSRequest
		err := json.Unmarshal(msg.Content, &req)
		if err != nil {
			log.Println("Failed during unmarshalling")
		}
		n.DNSRequestHandler(req, msg.Sender)

	case DNSResponse:
		var res BDNSResponse
		err := json.Unmarshal(msg.Content, &res)
		if err != nil {
			log.Println("Failed during unmarshalling")
		}
		n.DNSResponseHandler(res)

	case MsgTransaction:
		var tx blockchain.Transaction
		err := json.Unmarshal(msg.Content, &tx)
		if err != nil {
			log.Println("Failed during unmarshalling")
		}
		n.AddTransaction(&tx)

	case MsgBlock:
		var block blockchain.Block
		err := json.Unmarshal(msg.Content, &block)
		if err != nil {
			log.Println("Failed during unmarshalling")
		}
		n.AddBlock(&block)

	case MsgRandomNumber:
		var randomMsg RandomNumberMsg
		err := json.Unmarshal(msg.Content, &randomMsg)
		if err != nil {
			log.Println("Failed to unmarshal random number message:", err)
		}
		n.RandomNumberHandler(randomMsg.Epoch, hex.EncodeToString(randomMsg.Sender), randomMsg.SecretValue, randomMsg.RandomValue) // Store the received random number

	case MsgInv:
		n.HandleINV(msg.Sender)

	case MsgGetData:
		n.HandleGetData(msg.Sender)

	case MsgGetBlock:
		n.HandleGetBlock(msg.Sender)

	case MsgGetMerkle:
		n.HandleMerkleRequest(msg.Sender)
	}
}

// HandleGossip listens for messages from the gossip network
func (n *Node) HandleGossip() {
	for msg := range n.P2PNetwork.MsgChan {
		n.HandleMsgGivenType(msg)
	}

	fmt.Println("Exiting gossip listener for ", n.Address)
}

// Handles direct peer-to-peer messages
func (n *Node) ListenForDirectMessages() {
	// Handler for dns response
	n.P2PNetwork.Host.SetStreamHandler("/dns-response", func(s network.Stream) {
		defer s.Close()
		var msg GossipMessage
		if err := json.NewDecoder(s).Decode(&msg); err != nil {
			log.Println("Error decoding direct response:", err)
			return
		}

		n.HandleMsgGivenType(msg)
	})
}

func (n *Node) BroadcastTransaction(tx blockchain.Transaction) {
	n.AddTransaction(&tx)
	n.P2PNetwork.BroadcastMessage(MsgTransaction, tx)
}

func (n *Node) MakeDNSRequest(domainName string) {
	req := BDNSRequest{DomainName: domainName}
	n.P2PNetwork.BroadcastMessage(DNSRequest, req)
}

func (n *Node) BroadcastRandomNumber(epoch int64) {
	n.RegistryMutex.Lock()
	_, secretValues := consensus.CommitmentPhase(n.RegistryKeys)
	n.RegistryMutex.Unlock()
	
	nodeSecretValues := secretValues[hex.EncodeToString(n.KeyPair.PublicKey)]

	msg := RandomNumberMsg{
		Epoch:       epoch,
		SecretValue: nodeSecretValues.SecretValue,
		RandomValue: nodeSecretValues.RandomValue,
		Sender:      n.KeyPair.PublicKey,
	}
	n.RandomNumberHandler(epoch, hex.EncodeToString(n.KeyPair.PublicKey), nodeSecretValues.SecretValue, nodeSecretValues.RandomValue)
	n.P2PNetwork.BroadcastMessage(MsgRandomNumber, msg)
}

func (n *Node) DNSRequestHandler(req BDNSRequest, reqSender string) {
	// If this node is a light node, forward the query to a known full node
	if !n.IsFullNode {
		fmt.Println("Light node forwarding DNS query for:", req.DomainName)
		for _, peerID := range n.KnownFullPeers {
			if peerID != reqSender {
				n.P2PNetwork.DirectMessage(DNSRequest, req, peerID)
				fmt.Println(" Light node forwarded query to:", peerID)
				break
			}
		}
		return
	}

	// Otherwise, full node will handle it
	n.TxMutex.Lock()
	defer n.TxMutex.Unlock()
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
		n.P2PNetwork.DirectMessage(DNSResponse, res, reqSender)
	}
	fmt.Println("DNS Request received at ", n.Address, " -> ", req.DomainName)
}

func (n *Node) DNSResponseHandler(res BDNSResponse) {
	fmt.Println("DNS Response with Full node received at ", n.Address, " -> ", res.DomainName, " IP:", res.IP)
}

func (n *Node) RandomNumberHandler(epoch int64, sender string, secretValue int, randomValue int) {
	n.RandomMutex.Lock()
	defer n.RandomMutex.Unlock()

	if n.EpochRandoms[epoch] == nil {
		n.EpochRandoms[epoch] = make(map[string]consensus.SecretValues)
	}

	n.EpochRandoms[epoch][sender] = consensus.SecretValues{
		SecretValue: secretValue,
		RandomValue: randomValue,
	}
}
