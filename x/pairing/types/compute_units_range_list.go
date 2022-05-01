package types

import "encoding/json"

//reference: https://github.com/gogo/protobuf/blob/master/custom_types.md

func (cuListMessage CUListMessage) Equal(other CUListMessage) bool {
	return false //TODO implement
}

func (cuListMessage CUListMessage) Compare(other CUListMessage) int {
	return 0 //TODO implement
}

//func NewPopulatedT(r randyThetest) *T {}

func (cuListMessage CUListMessage) MarshalJSON() ([]byte, error) {
	return json.Marshal(cuListMessage)
}

func (cuListMessage *CUListMessage) UnmarshalJSON(data []byte) error {
	err := json.Unmarshal(data, &cuListMessage)
	if err != nil {
		return err
	}
	return nil
}
