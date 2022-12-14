package types

import "encoding/json"

//reference: https://github.com/gogo/protobuf/blob/master/custom_types.md

func (cuList StakeToMaxCUList) Equal(other StakeToMaxCUList) bool {
	if len(cuList.List) != len(other.List) {
		return false
	}

	for i, item := range cuList.List {
		if item != other.List[i] {
			return false
		}
	}

	return true
}

func (cuList StakeToMaxCUList) Compare(other StakeToMaxCUList) int {
	if len(cuList.List) != len(other.List) {
		return len(cuList.List) - len(other.List)
	}

	for i, item := range cuList.List {
		if item != other.List[i] {
			return (int(item.StakeThreshold.Amount.Int64()) - int(other.List[i].StakeThreshold.Amount.Int64())) + (int(item.MaxComputeUnits) - int(other.List[i].MaxComputeUnits))
		}
	}

	return 0
}

// func NewPopulatedT(r randyThetest) *T {}

func (cuList StakeToMaxCUList) MarshalJSON() ([]byte, error) {
	return json.Marshal(cuList.List)
}

func (cuList *StakeToMaxCUList) UnmarshalJSON(data []byte) error {
	err := json.Unmarshal(data, &cuList.List)
	if err != nil {
		return err
	}
	return nil
}
