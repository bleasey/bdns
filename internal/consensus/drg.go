package consensus

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"math/rand"
	"time"
)

type SecretValues struct {
	SecretValue int // uI -> revealed to other registries
	RandomValue int // rI -> kept as a secret
}

func CommitmentPhase(registryKeys [][]byte) (map[string]string, map[string]SecretValues) {
	commitments := make(map[string]string)
	secretValues := make(map[string]SecretValues)

	numRegistries := len(registryKeys)
	rand.New(rand.NewSource(time.Now().UnixNano()))

	for i := 0; i < numRegistries; i++ {
		uI := rand.Intn(1000) + 1
		rI := rand.Intn(1000) + 1

		data := fmt.Sprintf("%d%d", rI, uI)
		hash := sha256.Sum256([]byte(data))
		commitment := fmt.Sprintf("%x", hash) // stored in hexa

		// Store commitment
		commitments[hex.EncodeToString(registryKeys[i])] = commitment

		// Store secret values
		secretValues[hex.EncodeToString(registryKeys[i])] = SecretValues{
			SecretValue: uI,
			RandomValue: rI,
		}
	}

	return commitments, secretValues
}

func RevealPhase(secretValues map[string]SecretValues) map[string]int {
	revealedValues := make(map[string]int)

	for registry, values := range secretValues {
		revealedValues[registry] = values.SecretValue
	}

	return revealedValues
}

func RecoveryPhase(revealedValues map[string]int) float64 {
	seed := 0

	for _, secretValue := range revealedValues {
		seed ^= secretValue
	}
	normalizedSeed := float64(seed%1000) / 1000.0
	return normalizedSeed
}
