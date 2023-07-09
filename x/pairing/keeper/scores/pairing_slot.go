package scores

// object to hold the requirements for a specific slot (map of req names pointing to req object)
type PairingSlot struct {
	Reqs map[string]ScoreReq
}

func NewPairingSlot(reqs map[string]ScoreReq) *PairingSlot {
	slot := PairingSlot{
		Reqs: reqs,
	}
	return &slot
}

// generate a diff slot that contains the reqs that are in the slot receiver but not in the "other" slot
func (s PairingSlot) Diff(other *PairingSlot) *PairingSlot {
	reqsDiff := make(map[string]ScoreReq)
	for key := range s.Reqs {
		if _, found := other.Reqs[key]; !found {
			reqsDiff[key] = s.Reqs[key]
		}
	}

	return NewPairingSlot(reqsDiff)
}

// object to hold a slot and the number of times it's required
type PairingSlotGroup struct {
	Slot  *PairingSlot
	Count uint64
}

func NewPairingSlotGroup(slot *PairingSlot) *PairingSlotGroup {
	slotGroup := PairingSlotGroup{
		Slot:  slot,
		Count: 1,
	}
	return &slotGroup
}
