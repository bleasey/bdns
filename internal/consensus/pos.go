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
		stakeProbs := GetStakes(stakeData, index)

		prob := stakeProbs[registryStr]
		cumulativeProb += prob

		if seed <= cumulativeProb {
			return registry
		}
	}

	lastRegistry := registryKeys[len(registryKeys)-1]
	return lastRegistry
}

func GetStakes(stakeData map[string]int, index int) map[string]float64 {
	sum := 0.0

	// create a slice of the keys
	keys := make([]string, 0, len(stakeData))
	for k := range stakeData {
		keys = append(keys, k)
	}

	for i := index; i < len(keys); i++ {
		sum += float64(stakeData[keys[i]])
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

	for i := index; i < len(keys); i++ {
		registry := keys[i]
		stakeProbs[registry] = float64(stakeData[registry]) / sum
	}

	return stakeProbs
}
