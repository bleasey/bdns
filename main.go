package main

import (
	"fmt"
	"time"

	"github.com/bleasey/bdns/sims"
)

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

func main() {
	// sims.SimpleSim()
	sims.RandSim(numNodes, txTime, simulationTime, interval, slotInterval, slotsPerEpoch, seed, txProbability, queryProbability)
	if err := sims.CleanChainData(); err != nil {
		fmt.Printf("Warning: Error cleaning chaindata: %v\n", err)
	}
}
