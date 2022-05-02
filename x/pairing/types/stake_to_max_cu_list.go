package types

import "encoding/json"

//reference: https://github.com/gogo/protobuf/blob/master/custom_types.md

func (cuList StakeToMaxCUList) Equal(other StakeToMaxCUList) bool {
	return false //TODO implement?
}

func (cuList StakeToMaxCUList) Compare(other StakeToMaxCUList) int {
	return 0 //TODO implement?
}

//func NewPopulatedT(r randyThetest) *T {}

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
