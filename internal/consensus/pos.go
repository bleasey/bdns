package consensus

import (
	"encoding/hex"
)

func GetSlotLeaderUtil(registryKeys [][]byte, stakeData map[string]int, epochRandoms map[string]SecretValues) []byte {
	revealedValues := RevealPhase(epochRandoms)
	seed := RecoveryPhase(revealedValues)

	stakeProbs := GetStakes(stakeData)
	cumulativeProb := 0.0

	for _, registry := range registryKeys {
		registryStr := hex.EncodeToString(registry)
		prob := stakeProbs[registryStr]
		cumulativeProb += prob

		if seed <= cumulativeProb {
			return registry
		}
	}

	lastRegistry := registryKeys[len(registryKeys)-1]
	return lastRegistry
}

func GetStakes(stakeData map[string]int) map[string]float64 {
	sum := 0.0
	for _, numDomains := range stakeData {
		sum += float64(numDomains)
	}

	stakeProbs := make(map[string]float64)

	// If sum is 0, give equal probability to all registries
	if sum == 0 {
		equalProb := 1.0 / float64(len(stakeData))
		for registry := range stakeData {
			stakeProbs[registry] = equalProb
		}
		return stakeProbs
	}

	for registry, numDomains := range stakeData {
		stakeProbs[registry] = float64(numDomains) / sum
	}

	return stakeProbs
}
