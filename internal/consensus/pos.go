package consensus

import (
	"encoding/hex"
)

func GetSlotLeaderUtil(registryKeys [][]byte, stakeData map[string]int, epochRandoms map[string]SecretValues) []byte {
	revealedValues := RevealPhase(epochRandoms)
	seed := RecoveryPhase(revealedValues)
	cumulativeProb := 0.0

	for index, registry := range registryKeys {
		registryStr := hex.EncodeToString(registry)
		stakeProbs := GetStakes(index, registryKeys, stakeData)

		prob := stakeProbs[registryStr]
		cumulativeProb += prob

		if seed <= cumulativeProb {
			return registry
		}
	}

	lastRegistry := registryKeys[len(registryKeys)-1]
	return lastRegistry
}

func GetStakes(index int, registryKeys [][]byte, stakeData map[string]int) map[string]float64 {
	sum := 0.0

	for i := index; i < len(registryKeys); i++ {
		registry := hex.EncodeToString(registryKeys[i])
		sum += float64(stakeData[registry])
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

	for i := index; i < len(registryKeys); i++ {
		registry := hex.EncodeToString(registryKeys[i])
		stakeProbs[registry] = float64(stakeData[registry]) / sum
	}

	return stakeProbs
}
