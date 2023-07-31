package scores

// PairingSlot holds the set of requirements for a slot.
// It also holds the number of identical slots required for the pairing (count)
type PairingSlot struct {
	Reqs  map[string]ScoreReq
	Count int
}

func NewPairingSlot() *PairingSlot {
	return &PairingSlot{Count: 1}
}

// Subtract generates a diff slot that contains the reqs that are in the slot receiver but not in the "other" slot
func (s PairingSlot) Subtract(other *PairingSlot) *PairingSlot {
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

func (s PairingSlot) Equal(other *PairingSlot) bool {
	if len(s.Reqs) != len(other.Reqs) {
		return false
	}

	for key, req := range s.Reqs {
		otherReq, exists := other.Reqs[key]
		if !exists || !req.Equal(otherReq) {
			return false
		}
	}

	return true
}
