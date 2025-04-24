package network

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"time"

	"github.com/bleasey/bdns/internal/blockchain"
)

func InitializeP2PNodes(numNodes int, slotInterval int, slotsPerEpoch int, seed int) []*Node {
	ctx := context.Background()
	nodes := make([]*Node, numNodes)
	registryKeys := make([][]byte, numNodes)
	peerAddresses := []string{}
	topicName := "bdns-network"

	// Initialize nodes from ports range 4001 onwards
	for i := 0; i < numNodes; i++ {
		port := 4001 + i
		addr := fmt.Sprintf("/ip4/127.0.0.1/tcp/%d", port)
		isFull := i != 1 //  Only node 0 is light node
		node, err := NewNode(ctx, addr, topicName, isFull)
		if err != nil {
			log.Fatalf("Error creating node on port %d: %v", port, err)
		}

		peerAddr := fmt.Sprintf("%s/p2p/%s", addr, node.P2PNetwork.Host.ID().String())
		peerAddresses = append(peerAddresses, peerAddr)
		nodes[i] = node
		registryKeys[i] = node.KeyPair.PublicKey
	}

	// Set up peers for each node
	for _, node := range nodes {
		for _, addr := range peerAddresses {
			// Avoid self-conncection: check if port matches
			portStartIdx := len("/ip4/127.0.0.1/tcp/")
			portEndIdx := portStartIdx + 4
			if addr[portStartIdx:portEndIdx] == node.Address[portStartIdx:portEndIdx] {
				continue
			}

			err := node.P2PNetwork.ConnectToPeer(addr)
			if err != nil {
				fmt.Printf("Failed to connect to peer %s: %v\n", addr, err)
			}
		}
	}

	currTimestamp := time.Now().Unix()

	// Initialize nodes given registryKeys and params
	for i := 0; i < numNodes; i++ {
		go nodes[i].InitializeNodeAsync(strconv.Itoa(i), registryKeys, currTimestamp, int64(slotInterval), int64(slotsPerEpoch), float64(seed))
	}

	time.Sleep(2 * time.Second) // Let the network stabilize

	fmt.Printf("Nodes initialized as peers, listening on localhost:4001 to localhost:400%d.\n", numNodes)
	fmt.Println("- - - - - - - - - - - -")

	// Collect full node peer IDs
	fullPeerIDs := []string{}
	for _, node := range nodes {
		if node.IsFullNode {
			fullPeerIDs = append(fullPeerIDs, node.PeerID)
		}
	}

	// Assign known full peers to every node
	for _, node := range nodes {
		node.KnownFullPeers = fullPeerIDs
	}
	return nodes
}

func (n *Node) InitializeNodeAsync(chainID string, registryKeys [][]byte, initialTimestamp int64, slotInterval int64, slotsPerEpoch int64, seed float64) {
	initialWaitTime := int64(5) // wait for initial stability (in secs)
	n.RegistryKeys = registryKeys
	n.Blockchain = blockchain.CreateBlockchain(chainID)
	n.Config.InitialTimestamp = initialTimestamp + initialWaitTime
	n.Config.SlotInterval = slotInterval
	n.Config.SlotsPerEpoch = slotsPerEpoch
	n.Config.EpochsPerVote = 2
	n.Config.VoteRevokeCuttoff = 0.6
	n.Config.Seed = seed
	n.Config.RegistryUpdateAfter = 3 // 3 epochs
	n.Config.BlockTxLimit = 10

	fmt.Printf("Node %s initialized with chain ID %s\n", n.Address, chainID)
	time.Sleep(time.Duration(initialWaitTime) * time.Second) // wait till end of epoch
	n.CreateBlockIfLeader()
}

func NodesCleanup(nodes []*Node) {
	fmt.Println("- - - - - - - - - - - -")
	fmt.Println("Cleaning up nodes....")

	// Close all databases
	for _, node := range nodes {
		if err := node.Blockchain.CloseDB(); err != nil {
			log.Printf("Failed to close database for node %s: %v", node.Address, err)
		}

		node.P2PNetwork.Close()
	}

	// Wait briefly to ensure file handles are released
	time.Sleep(2 * time.Second)

	fmt.Println("Simulation completed.")
}
