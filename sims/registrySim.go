package sims

import (
	"fmt"
	"sync"
	"time"

	"github.com/bleasey/bdns/internal/blockchain"
	"github.com/bleasey/bdns/internal/network"
)

func RegistrySim() {
	const (
		numNodes       = 10
		slotInterval   = 5
		slotsPerEpoch  = 2
		seed           = 42
		waitEpochs     = 4
	)

	var wg sync.WaitGroup
	nodes := network.InitializeP2PNodes(numNodes, slotInterval, slotsPerEpoch, seed)

	fmt.Println("Waiting for genesis block to be created...\n")
	time.Sleep(time.Duration(slotInterval) * time.Second)

	// Pick node 0 to add a new registry via transaction
	creator := nodes[0]
	newKeyPair := blockchain.NewKeyPair()

	// Create and broadcast registry-add transaction
	regTx := blockchain.NewRegistryTransaction(
		blockchain.REGISTRY_REGISTER,
		newKeyPair.PublicKey,
		nil,
		creator.KeyPair.PublicKey,
		&creator.KeyPair.PrivateKey,
		creator.TransactionPool,
	)
	creator.BroadcastTransaction(*regTx)

	// Wait for N epochs to let the registry propagate
	totalWait := time.Duration(waitEpochs*slotsPerEpoch*slotInterval) * time.Second
	fmt.Printf("Waiting %d epochs (~%v) to ensure propagation...\n\n", waitEpochs, totalWait)
	time.Sleep(totalWait)

	// Check all nodes to confirm new registry was added
	foundCount := 0
	for _, node := range nodes {
		for _, reg := range node.RegistryKeys {
			if string(reg) == string(newKeyPair.PublicKey) {
				fmt.Printf("Node %s includes the new registry.\n", node.Address)
				foundCount++
				break
			}
		}
	}
	fmt.Printf("\n%d/%d nodes have the new registry.\n", foundCount, numNodes)

	time.Sleep(totalWait * 10)
	wg.Wait()
	network.NodesCleanup(nodes)
}
