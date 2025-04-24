package sims

import (
	"fmt"
	"math/rand"
	"sync"
	"time"

	"github.com/bleasey/bdns/internal/blockchain"
	"github.com/bleasey/bdns/internal/network"
	"github.com/bleasey/bdns/client"
)

func RandSim() {
	const (
		numNodes         = 10
		txTime           = 20 * time.Second
		simulationTime   = 60 * time.Second
		interval         = 1 * time.Second
		slotInterval     = 5
		slotsPerEpoch    = 2
		seed             = 0
		txProbability    = 0.05
		queryProbability = 0.02
	)

	var wg sync.WaitGroup
	nodes := network.InitializeP2PNodes(numNodes, slotInterval, slotsPerEpoch, seed)

	fmt.Println("Waiting for genesis block to be created...\n")
	time.Sleep(time.Duration(slotInterval) * time.Second)

	domains := make([]string, 0) // list of registered domains
	txOnlyTime := time.Now().Add(txTime)
	simStopTime := time.Now().Add(simulationTime)
	wg.Add(numNodes)

	for i, node := range nodes {
		go func(node *network.Node, id int) {
			defer wg.Done()

			for time.Now().Before(simStopTime) {
				// Chance to create transaction
				if rand.Float64() < txProbability {
					domain := fmt.Sprintf("tx%d-node%d.com", len(domains), id+1)
					ip := fmt.Sprintf("10.0.%d.%d", id+1, rand.Intn(255))
					ttl := int64(3600)
					tx := blockchain.NewTransaction(blockchain.REGISTER, domain, ip, ttl, node.KeyPair.PublicKey, &node.KeyPair.PrivateKey, node.TransactionPool)
					node.BroadcastTransaction(*tx)
					fmt.Printf("Node %d sent transaction for domain %s\n", id+1, domain)
					domains = append(domains, domain) // assuming for simplicity, the tx was accepted
				}

				// Chance to send a DNS request
				if time.Now().After(txOnlyTime) && rand.Float64() < queryProbability {
					target := rand.Intn(len(domains))
					domain := domains[target]
					// fmt.Printf("Node %d querying for %s\n", id+1, domain)
					node.MakeDNSRequest(domain)
				}

				time.Sleep(interval)
			}
		}(node, i)
	}


	wg.Wait()
	time.Sleep(10 * time.Second) // wait until nodes are ready
    client.RunAutoClient(domains)
	network.NodesCleanup(nodes)
}
