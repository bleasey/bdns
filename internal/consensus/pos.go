package consensus

import "sort"

func GetSlotLeader(epoch int, seed int, stakeData map[string]int) []byte {
	registeredKeys := make([]string, 0, len(stakeData))
	for key := range stakeData {
		registeredKeys = append(registeredKeys, key)
	}

	// Sort the keys
	sort.Strings(registeredKeys)

	// For now, sequentially choose the slot leader from the registered keys
	n := len(registeredKeys)
	slotLeaderIndex := epoch % n
	return []byte(registeredKeys[slotLeaderIndex])
}	
