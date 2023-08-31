package format

import (
	"github.com/lavanet/lava/utils"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
)

const (
	IDFieldName    = "id"
	DefaultIDValue = 1
)

// removing ID from the request so we have cache matches when id is changing
func FormatterForRelayRequestAndResponseJsonRPC() (inputFormatter func([]byte) []byte, outputFormatter func([]byte) []byte) {
	// extractedID is used in outputFormatter to return the id that the user requested
	var extractedID int64 = -1
	inputFormatter = func(inpData []byte) []byte {
		if len(inpData) < 3 {
			return inpData
		}

		result := gjson.GetBytes(inpData, IDFieldName)
		if result.Type != gjson.Number {
			// something is wrong with the id field, its not a number
			_ = utils.LavaFormatError("invalid field type for id in json formatter", nil, utils.Attribute{Key: "jsonData", Value: inpData})
			return inpData
		}
		extractedID = result.Int()
		modifiedInp, err := sjson.SetBytes(inpData, IDFieldName, DefaultIDValue)
		if err != nil {
			_ = utils.LavaFormatError("failed to set id 1 in json", nil, utils.Attribute{Key: "jsonData", Value: inpData})
			return inpData
		}
		return modifiedInp
	}
	outputFormatter = func(inpData []byte) []byte {
		if len(inpData) == 0 {
			return inpData
		}

		result := gjson.GetBytes(inpData, IDFieldName)
		if result.Type != gjson.Number {
			// something is wrong with the id field, its not a number
			return inpData
		}
		modifiedInp, err := sjson.SetBytes(inpData, IDFieldName, extractedID)
		if err != nil {
			return inpData
		}
		return modifiedInp
	}
	return inputFormatter, outputFormatter
}
