package scores

import "strings"

// PairingSlot holds the set of requirements for a slot.
// It also holds the number of identical slots required for the pairing (count)
type PairingSlot struct {
	Reqs  map[string]ScoreReq
	Count uint64
}

func NewPairingSlot() *PairingSlot {
	return &PairingSlot{Count: 1}
}

// Subtract generates a diff slot that contains the reqs that are in the slot receiver but not in the "other" slot
func (s PairingSlot) Subtract(other *PairingSlot) *PairingSlot {
	reqsDiff := make(map[string]ScoreReq)
	for key := range s.Reqs {
		if _, found := other.Reqs[key]; !found {
			reqsDiff[key] = s.Reqs[key]
		}
	}

	diffSlot := NewPairingSlot()
	diffSlot.Reqs = reqsDiff
	return diffSlot
}

// GetSlotKey generates a unique key of the slot based on its requirements
func (s PairingSlot) GetSlotKey() string {
	key := ""
	for reqName := range s.Reqs {
		key += reqName + "-"
	}

	return strings.TrimSuffix(key, "-")
}
