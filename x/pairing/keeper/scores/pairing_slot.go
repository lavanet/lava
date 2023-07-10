package scores

// PairingSlot holds the set of requirements for a slot
type PairingSlot struct {
	Reqs map[string]ScoreReq
}

func NewPairingSlot(reqs map[string]ScoreReq) *PairingSlot {
	return &PairingSlot{Reqs: reqs}
}

// Subtract generates a diff slot that contains the reqs that are in the slot receiver but not in the "other" slot
func (s PairingSlot) Subtract(other *PairingSlot) *PairingSlot {
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
