package network

import (
	"encoding/json"
	"log"
	"net"
    "sync"
    "bufio"

	"github.com/bleasey/bdns/internal/blockchain"
    "github.com/bleasey/bdns/internal/index"
)

// Represents a peer in the blockchain network
type Node struct {
    Address   			string
    KeyPair		        *blockchain.KeyPair
    RegistryKeys        [][]byte
    Peers     			map[string]net.Conn // ip to connection
    PeersMutex  		sync.Mutex
	TransactionPool 	map[int]*blockchain.Transaction
    TxMutex 			sync.Mutex
    IndexManager        *index.IndexManager
    Blockchain 			*blockchain.Blockchain // Reference to the blockchain
	BcMutex				sync.Mutex
}

// Initializes a new P2P node
func NewNode(address string) *Node {
    return &Node{
        Address: address,
        KeyPair: blockchain.NewKeyPair(),
        Peers: make(map[string]net.Conn),
        TransactionPool: make(map[int]*blockchain.Transaction),
        IndexManager: index.NewIndexManager(),
        Blockchain: nil,
    }
}

func (n *Node) InitializeNodeAsync(chainID string, registryKeys [][]byte, randomness []byte, epochInterval int, seed int) {
    n.RegistryKeys = registryKeys
    n.Blockchain = blockchain.CreateBlockchain(chainID, registryKeys, randomness)
    go n.Start()
    go n.CreateBlockIfLeader(epochInterval, seed)
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

    log.Printf("Node listening on %s", n.Address)

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
        json.Unmarshal(msg.Data, &req)
        n.DNSRequestHandler(req, conn)

    case DNSResponse:
        var res BDNSResponse
        json.Unmarshal(msg.Data, &res)
        n.DNSResponseHandler(res, conn)

    case MsgTransaction:
        var tx blockchain.Transaction
        json.Unmarshal(msg.Data, &tx)
        n.AddTransaction(&tx)

    case MsgBlock:
        var block blockchain.Block
        json.Unmarshal(msg.Data, &block)
        n.AddBlock(&block)
        
    case MsgChainRequest:
        n.Blockchain.SendBlockchain(conn)

    case MsgChainResponse:
        n.Blockchain.ReplaceChain(conn, &n.BcMutex)

    case MsgPeerRequest:
		return
        // n.SendPeers(conn)

    case MsgPeerResponse:
		return
        // var peers []string
        // json.Unmarshal(msg.Data, &peers)
        // for _, peer := range peers {
        //     n.ConnectToPeer(peer)
        // }
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

func (n *Node) SendToPeer(conn net.Conn , msg Message) {
    /*
    Broadcasts msg for now
    TODO: Modify to send to a specific peer
    */
    n.PeersMutex.Lock()
    defer n.PeersMutex.Unlock()
    
    for _, conn := range n.Peers {
        _, err := conn.Write(append(msg.Encode(), '\n')) // appending the delimiter to msg
        if err != nil {
            log.Println("Error broadcasting message:", err)
        }
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
