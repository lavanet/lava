package scores

// PairingSlot holds the set of requirements for a slot.
// It also holds the number of identical slots required for the pairing (count)

type PairingSlotInf interface {
	Requirements() map[string]ScoreReq
}

type PairingSlotGroup struct {
	Reqs  map[string]ScoreReq
	Count int
}

func (s PairingSlotGroup) Requirements() map[string]ScoreReq {
	return s.Reqs
}

func NewPairingSlotGroup(pairingSlot *PairingSlot) *PairingSlotGroup {
	return &PairingSlotGroup{Count: 1, Reqs: pairingSlot.Reqs}
}

// Subtract generates a diff slot that contains the reqs that are in the slot receiver but not in the "other" slot
func (s PairingSlotGroup) Subtract(other *PairingSlotGroup) *PairingSlot {
	reqsDiff := make(map[string]ScoreReq)
	for key := range s.Reqs {
		req := s.Reqs[key]
		otherReq, found := other.Reqs[key]
		if !found {
			reqsDiff[key] = req
		} else if !req.Equal(otherReq) {
			reqsDiff[key] = req
		}
	}

	diffSlot := NewPairingSlot()
	diffSlot.Reqs = reqsDiff
	return diffSlot
}

type PairingSlot struct {
	Reqs map[string]ScoreReq
}

func (s PairingSlot) Requirements() map[string]ScoreReq {
	return s.Reqs
}

func NewPairingSlot() *PairingSlot {
	return &PairingSlot{
		Reqs: map[string]ScoreReq{},
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
