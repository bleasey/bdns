package sims

import (
	"fmt"
	"math/rand"
	"sync"
	"time"

	"github.com/bleasey/bdns/internal/blockchain"
	"github.com/bleasey/bdns/internal/metrics"
	"github.com/bleasey/bdns/internal/network"
)

func SimpleSim() {
	// Constants
	const numNodes = 6
	const slotInterval = 5
	const slotsPerEpoch = 2
	const seed = 0
	var wg sync.WaitGroup

	metrics := metrics.GetDNSMetrics()

	nodes := network.InitializeP2PNodes(numNodes, slotInterval, slotsPerEpoch, seed)

	fmt.Println("Waiting for genesis block to be created...")
	time.Sleep(time.Duration(slotInterval) * time.Second)

	// Each node registers its own domains
	for i, node := range nodes {
		domainName := fmt.Sprintf("node%d.com", i+1)
		ip := fmt.Sprintf("192.168.1.%d", i+1)
		ttl := int64(3600)
		tx := blockchain.NewTransaction(blockchain.REGISTER, domainName, ip, ttl, node.KeyPair.PublicKey, &node.KeyPair.PrivateKey, node.TransactionPool)
		node.BroadcastTransaction(*tx)
		fmt.Printf("Node %d sent transaction for domain %s\n", i+1, tx.DomainName)
	}

	fmt.Printf("Waiting for end of epoch for block creation...")
	time.Sleep(slotInterval * slotsPerEpoch * time.Second) // Let transactions propagate via block from first epoch

	// Periodic querying simulation
	wg.Add(numNodes)
	for i, node := range nodes {
		if i != 0 {
			wg.Done()
			continue
		}

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

				node.MakeDNSRequest(domain, metrics)

				time.Sleep(time.Duration(slotInterval * time.Second))
			}
		}(node, i)
	}
	wg.Wait()                   // Wait for queries to complete
	network.NodesCleanup(nodes) // Cleanup chaindata directory
}
