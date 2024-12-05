package scores

// PairingSlot holds the set of requirements for a slot.
// It also holds the number of identical slots required for the pairing (count)

type PairingSlotInf interface {
	Requirements() map[string]ScoreReq
}

type PairingSlotGroup struct {
	Reqs    map[string]ScoreReq
	indexes []int
}

func (s PairingSlotGroup) Indexes() []int {
	return s.indexes
}

func (s PairingSlotGroup) Requirements() map[string]ScoreReq {
	return s.Reqs
}

func NewPairingSlotGroup(pairingSlot *PairingSlot) *PairingSlotGroup {
	return &PairingSlotGroup{indexes: []int{pairingSlot.Index}, Reqs: pairingSlot.Reqs}
}

func (psg *PairingSlotGroup) AddToGroup(pairingSlot *PairingSlot) {
	psg.indexes = append(psg.indexes, pairingSlot.Index)
}

// Subtract generates a diff slot that contains the reqs that are in the slot receiver but not in the "other" slot
func (psg PairingSlotGroup) Subtract(other *PairingSlotGroup) *PairingSlot {
	reqsDiff := make(map[string]ScoreReq)
	for key := range psg.Reqs {
		req := psg.Reqs[key]
		otherReq, found := other.Reqs[key]
		if !found {
			reqsDiff[key] = req
		} else if req != nil && !req.Equal(otherReq) {
			reqsDiff[key] = req
		}
	}

	diffSlot := NewPairingSlot(-1) // we set new.Subtract(old)
	diffSlot.Reqs = reqsDiff
	return diffSlot
}

type PairingSlot struct {
	Reqs  map[string]ScoreReq
	Index int
}

func (s PairingSlot) Requirements() map[string]ScoreReq {
	return s.Reqs
}

func NewPairingSlot(slotIndex int) *PairingSlot {
	return &PairingSlot{
		Reqs:  map[string]ScoreReq{},
		Index: slotIndex,
	}
}

func (s PairingSlot) Equal(other PairingSlotInf) bool {
	if len(s.Reqs) != len(other.Requirements()) {
		return false
	}

	for key, req := range s.Reqs {
		otherReq, exists := other.Requirements()[key]
		if !exists || !req.Equal(otherReq) {
			return false
		}
	}

	return true
}
