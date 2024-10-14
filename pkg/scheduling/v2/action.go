package v2

type action struct {
	lastReplenishedSlotCount   int
	lastReplenishedWorkerCount int

	// note that slots can be used across multiple actions, hence the pointer
	slots []*slot
}

func (a *action) activeCount() int {
	count := 0

	for _, slot := range a.slots {
		if slot.active() {
			count++
		}
	}

	return count
}
