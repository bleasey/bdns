package consensus

func GetSlotLeader(epoch int, seed int, registryKeys [][]byte, stakeData map[string]int) []byte {
	// For now, sequentially choose the slot leader from registryKeys
	n := len(registryKeys)
	slotLeaderIndex := epoch % n
	return registryKeys[slotLeaderIndex]
}	
