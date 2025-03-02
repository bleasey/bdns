package consensus

func GetSlotLeader(epoch int, _ int, registryKeys [][]byte, _ map[string]int) []byte {
	// For now, sequentially choose the slot leader from registryKeys, parameters are the seed, stakedata
	n := len(registryKeys)
	slotLeaderIndex := epoch % n
	return registryKeys[slotLeaderIndex]
}
