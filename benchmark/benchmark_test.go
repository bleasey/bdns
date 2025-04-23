package benchmark

import (
	"fmt"
	"testing"
	"time"

	"github.com/bleasey/bdns/sims"
)

// TestCase defines the parameters for a network simulation
type TestCase struct {
	Name             string
	NumNodes         int
	TransactionTime  time.Duration
	SimulationTime   time.Duration
	Interval         time.Duration
	SlotInterval     int
	SlotsPerEpoch    int
	Seed             int
	TxProbability    float64
	QueryProbability float64
}

var testCases = []TestCase{
	// {
	// 	Name:             "Small Network",
	// 	NumNodes:         10,
	// 	TransactionTime:  20 * time.Second,
	// 	SimulationTime:   60 * time.Second,
	// 	Interval:         500 * time.Millisecond,
	// 	SlotInterval:     5,
	// 	SlotsPerEpoch:    2,
	// 	Seed:             0,
	// 	TxProbability:    0.2,
	// 	QueryProbability: 0.3,
	// },
	{
		Name:             "Medium Network",
		NumNodes:         25,
		TransactionTime:  20 * time.Second,
		SimulationTime:   120 * time.Second,
		Interval:         500 * time.Millisecond,
		SlotInterval:     5,
		SlotsPerEpoch:    2,
		Seed:             0,
		TxProbability:    0.2,
		QueryProbability: 0.3,
	},
	// {
	// 	Name:             "Large Network",
	// 	NumNodes:         50,
	// 	TransactionTime:  20 * time.Second,
	// 	SimulationTime:   120 * time.Second,
	// 	Interval:         500 * time.Millisecond,
	// 	SlotInterval:     5,
	// 	SlotsPerEpoch:    2,
	// 	Seed:             0,
	// 	TxProbability:    0.2,
	// 	QueryProbability: 0.3,
	// },
}

func TestNetworks(t *testing.T) {
	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			fmt.Printf("\n=== Running %s Test ===\n", tc.Name)

			if err := sims.CleanChainData(); err != nil {
				t.Fatalf("Failed to clean chaindata: %v", err)
			}
			time.Sleep(5 * time.Second)

			sims.RandSim(
				tc.NumNodes,
				tc.TransactionTime,
				tc.SimulationTime,
				tc.Interval,
				tc.SlotInterval,
				tc.SlotsPerEpoch,
				tc.Seed,
				tc.TxProbability,
				tc.QueryProbability,
			)

			// Add a delay between test cases to ensure clean separation
			time.Sleep(5 * time.Second)
		})
	}
}
