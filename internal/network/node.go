package network

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/bleasey/bdns/internal/blockchain"
	"github.com/bleasey/bdns/internal/index"
)

// Represents a peer in the blockchain network
type Node struct {
	Address         string
	Config          NodeConfig
	KeyPair         *blockchain.KeyPair
	RegistryKeys    [][]byte
	SlotLeaders     map[int64][]byte // epoch to slot leader
	SlotMutex       sync.Mutex
	Peers           map[string]net.Conn // ip to connection
	PeersMutex      sync.Mutex
	TransactionPool map[int]*blockchain.Transaction
	TxMutex         sync.Mutex
	IndexManager    *index.IndexManager
	Blockchain      *blockchain.Blockchain // Reference to the blockchain
	BcMutex         sync.Mutex
}

// Node Config
type NodeConfig struct {
	InitialTimestamp int64
	EpochInterval    int64
	Seed             int64
}

// Initializes a new P2P node
func NewNode(address string) *Node {
	return &Node{
		Address:         address,
		KeyPair:         blockchain.NewKeyPair(),
		SlotLeaders:     make(map[int64][]byte),
		Peers:           make(map[string]net.Conn),
		TransactionPool: make(map[int]*blockchain.Transaction),
		IndexManager:    index.NewIndexManager(),
		Blockchain:      nil,
	}
}

func (n *Node) InitializeNodeAsync(chainID string, registryKeys [][]byte, initialTimestamp int64, epochInterval int64, seed int64) {
	n.RegistryKeys = registryKeys
	n.Blockchain = blockchain.CreateBlockchain(chainID)
	n.Config.InitialTimestamp = initialTimestamp + epochInterval // First block is created 'epoch' secs after initialization
	n.Config.EpochInterval = epochInterval
	n.Config.Seed = seed
	go n.Start()
	go n.CreateBlockIfLeader(epochInterval)
}

// Adds a peer to the list
func (n *Node) AddPeer(address string, conn net.Conn) {
	n.PeersMutex.Lock()
	defer n.PeersMutex.Unlock()
	n.Peers[address] = conn
}

// Removes a peer from the list
func (n *Node) RemovePeer(address string) {
	n.PeersMutex.Lock()
	defer n.PeersMutex.Unlock()
	delete(n.Peers, address)
}

// Starts the node's server to listen for incoming connections
func (n *Node) Start() {
	listener, err := net.Listen("tcp", n.Address)
	if err != nil {
		log.Fatalf("Failed to start node: %v", err)
	}
	defer listener.Close()

	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Println("Connection error:", err)
			continue
		}
		go n.HandleConnection(conn)
	}
}

// Processes incoming messages from peers
func (n *Node) HandleConnection(conn net.Conn) {
	defer conn.Close()
	address := conn.RemoteAddr().String()
	n.AddPeer(address, conn)

	reader := bufio.NewReader(conn)
	for {
		messageData, err := reader.ReadBytes('\n') // Read until newline
		if err != nil {
			log.Printf("Error reading from %s: %v", address, err)
			break
		}

		msg, err := DecodeMessage(messageData)
		if err != nil {
			log.Printf("Invalid message from %s", address)
			continue
		}

		n.ProcessMessage(msg, conn)
	}

	n.RemovePeer(address)
}

// Handles messages based on their type
func (n *Node) ProcessMessage(msg *Message, conn net.Conn) {
	switch msg.Type {
	case DNSRequest:
		var req BDNSRequest
		err := json.Unmarshal(msg.Data, &req)
		if err != nil {
			log.Println("Failed during unmarshalling")
		}
		n.DNSRequestHandler(req, conn)

	case DNSResponse:
		var res BDNSResponse
		err := json.Unmarshal(msg.Data, &res)
		if err != nil {
			log.Println("Failed during unmarshalling")
		}
		n.DNSResponseHandler(res, conn)

	case MsgTransaction:
		var tx blockchain.Transaction
		err := json.Unmarshal(msg.Data, &tx)
		if err != nil {
			log.Println("Failed during unmarshalling")
		}
		n.AddTransaction(&tx)

	case MsgBlock:
		var block blockchain.Block
		err := json.Unmarshal(msg.Data, &block)
		if err != nil {
			log.Println("Failed during unmarshalling")
		}
		n.AddBlock(&block)

	case MsgChainRequest:
		n.Blockchain.SendBlockchain(conn)

	case MsgChainResponse:
		n.Blockchain.ReplaceChain(conn, &n.BcMutex)
	}
}

// Sends a message to all connected peers
func (n *Node) Broadcast(msg Message) {
	n.PeersMutex.Lock()
	defer n.PeersMutex.Unlock()

	for _, conn := range n.Peers {
		_, err := conn.Write(append(msg.Encode(), '\n')) // appending the delimiter to msg
		if err != nil {
			log.Println("Error broadcasting message:", err)
		}
	}
}

// Send a message to a specific peer
func (n *Node) SendToPeer(conn net.Conn, msg Message) {
	n.PeersMutex.Lock()
	defer n.PeersMutex.Unlock()

	// fmt.Println("Sending to: ", conn.RemoteAddr(), " ", conn.LocalAddr())

	_, err := conn.Write(append(msg.Encode(), '\n')) // appending the delimiter to msg
	if err != nil {
		log.Println("Error sending message to peer:", err)
	}
}

// Connects to a new peer
func (n *Node) ConnectToPeer(address string) {
	conn, err := net.Dial("tcp", address)
	if err != nil {
		log.Printf("Failed to connect to peer %s", address)
		return
	}

	n.AddPeer(address, conn)
}

// Get peers connected to each other for a simulation
func InitializeNodesAsPeers(numNodes int, epochInterval int, seed int) ([]*Node, []string, [][]byte) {
	// Create and start nodes
	nodes := make([]*Node, numNodes)
	registryKeys := make([][]byte, numNodes)

	// Set node addresses and keys
	nodeAddresses := []string{}
	for i := 0; i < numNodes; i++ {
		nodeAddress := fmt.Sprintf("localhost:500%d", i)
		nodeAddresses = append(nodeAddresses, nodeAddress)
		nodes[i] = NewNode(nodeAddress)
		registryKeys[i] = nodes[i].KeyPair.PublicKey
	}

	currTimestamp := time.Now().Unix()

	// Initialize nodes given registryKeys and params
	for i := 0; i < numNodes; i++ {
		nodes[i].InitializeNodeAsync(strconv.Itoa(i), registryKeys, currTimestamp, int64(epochInterval), int64(seed))
	}

	time.Sleep(2 * time.Second) // Let the network stabilize

	// Connect nodes to each other
	for i, node := range nodes {
		for j, addr := range nodeAddresses {
			if i != j {
				node.ConnectToPeer(addr)
			}
		}
	}

	fmt.Printf("Nodes initialized as peers, listening on localhost:5000 to localhost:500%d.\n", numNodes-1)
	fmt.Println("- - - - - - - - - - - -")

	return nodes, nodeAddresses, registryKeys
}

func NodesCleanup(nodes []*Node) {
	fmt.Println("- - - - - - - - - - - -")
	fmt.Println("Cleaning up nodes....")

	// Close all databases
	for _, node := range nodes {
		if err := node.Blockchain.CloseDB(); err != nil {
			log.Printf("Failed to close database for node %s: %v", node.Address, err)
		}
	}

	// Wait briefly to ensure file handles are released
	time.Sleep(2 * time.Second)

	// Delete chaindata directory
	dir := "chaindata"
	if err := os.RemoveAll(dir); err != nil {
		log.Fatalf("Failed to clear directory %s: %v", dir, err)
	}
	fmt.Println("Simulation completed.")
}
