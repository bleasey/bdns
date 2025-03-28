package consensus

import (
	"fmt"
)

func GetSlotLeaderUtil(epoch int, registryKeys [][]byte, stakeData map[string]int) []byte {
	if (epoch == 0) {
		return registryKeys[0]
	}
	_, secretValues := CommitmentPhase(registryKeys)
    revealedValues := RevealPhase(secretValues)
    seed := RecoveryPhase(revealedValues)

	for _, val := range stakeData {
		fmt.Println("StakeValue-> ", val)
	}
	
	stakeProbs := GetStakes(stakeData)
	cumulativeProb := 0.0

	for _, registry := range registryKeys {
		cumulativeProb += stakeProbs[string(registry)]
		if seed <= cumulativeProb {
			return registry
		}
	}

	return registryKeys[len(registryKeys)-1]
}

func GetStakes(stakeData map[string]int) map[string]float64 {
	sum := 0.0
	for _, numDomains := range stakeData {
		sum += float64(numDomains)
	}

	stakeProbs := make(map[string]float64)
	for registry, numDomains := range stakeData {
		stakeProbs[registry] = float64(numDomains) / sum
	}

	return stakeProbs
}
