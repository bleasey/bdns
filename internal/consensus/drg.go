package consensus

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"math/rand"
	"time"
)

type SecretValues struct {
	SecretValue int // u_i -> revealed to other registries
	RandomValue int // r_i -> kept as a secret
}

func CommitmentPhase(registryKeys [][]byte) (map[string]string, map[string]SecretValues) {
	commitments := make(map[string]string)
	secretValues := make(map[string]SecretValues)

	numRegistries := len(registryKeys)
	rand.Seed(time.Now().UnixNano())

	for i := 0; i < numRegistries; i++ {
		u_i := rand.Intn(1000) + 1
		r_i := rand.Intn(1000) + 1

		data := fmt.Sprintf("%d%d", r_i, u_i)
		hash := sha256.Sum256([]byte(data))
		commitment := fmt.Sprintf("%x", hash) // stored in hexa

		// Store commitment
		commitments[hex.EncodeToString(registryKeys[i])] = commitment

		// Store secret values
		secretValues[hex.EncodeToString(registryKeys[i])] = SecretValues{
			SecretValue: u_i,
			RandomValue: r_i,
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
