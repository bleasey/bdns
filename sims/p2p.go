package sims

import (
	"fmt"
	"math/rand"
	"strconv"
	"sync"
	"time"

	"github.com/bleasey/bdns/internal/blockchain"
	"github.com/bleasey/bdns/internal/network"
)

func SimpleSim1() {
	// Constants
	const numNodes = 4
	const epochInterval = 10
	const seed = 0
	var wg sync.WaitGroup

	// Create and start nodes
	nodes := make([]*network.Node, numNodes)
	registryKeys := make([][]byte, numNodes)
	nodeAddresses := []string{"localhost:5001", "localhost:5002", "localhost:5003", "localhost:5004"}

	for i := 0; i < numNodes; i++ {
		nodes[i] = network.NewNode(nodeAddresses[i])
		registryKeys[i] = nodes[i].KeyPair.PublicKey
	}

	for i := 0; i < numNodes; i++ {
		nodes[i].InitializeNodeAsync(strconv.Itoa(i), registryKeys, []byte("randomness"), epochInterval, seed)
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

	// Each node registers its own domains
	for i, node := range nodes {
		tx := blockchain.Transaction{
			TID:        rand.Intn(1_000_000),
			Type:       blockchain.REGISTER,
			Timestamp:  time.Now().Unix(),
			DomainName: fmt.Sprintf("node%d.com", i+1),
			IP:         fmt.Sprintf("192.168.1.%d", i+1),
			TTL:        3600,
			OwnerKey:   node.KeyPair.PublicKey,
		}
		node.BroadcastTransaction(tx)
		fmt.Printf("Node %d registered domain %s\n", i+1, tx.DomainName)
	}

	fmt.Println("Waiting for end of epoch for block creation....")
	time.Sleep(epochInterval * time.Second) // Let transactions propagate via block from first epoch

	// Periodic querying simulation
	wg.Add(numNodes)
	for i, node := range nodes {
		go func(_ *network.Node, id int) {
			defer wg.Done()
			for j := 0; j < 1; j++ {
				// Randomly pick a domain to query
				queryNode := rand.Intn(numNodes)
				if queryNode == id {
					queryNode = (queryNode + 1) % numNodes
				}

				domain := fmt.Sprintf("node%d.com", queryNode+1)
				fmt.Printf("Node %d querying %s\n", id+1, domain)

				node.MakeDNSRequest(domain)

				time.Sleep(time.Duration(20 * time.Second))
			}
		}(node, i)
	}

	wg.Wait() // Wait for queries to complete

	fmt.Println("Simulation completed.")
}
