package cacheformat

import (
	"encoding/json"

	"github.com/lavanet/lava/v5/utils"
	spectypes "github.com/lavanet/lava/v5/types/spec"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
)

const (
	IDFieldName    = "id"
	DefaultIDValue = 1
)

// FormatterForRelayRequestAndResponse returns input and output formatter functions for the given
// apiInterface. The input formatter normalizes the request for cache key generation (e.g. strips the
// JSON-RPC id field). The output formatter restores the original id in the response.
func FormatterForRelayRequestAndResponse(apiInterface string) (inputFormatter func([]byte) []byte, outputFormatter func([]byte) []byte) {
	switch apiInterface {
	case spectypes.APIInterfaceJsonRPC, spectypes.APIInterfaceTendermintRPC:
		return FormatterForRelayRequestAndResponseJsonRPC()
	default:
		return IdentityFormatter()
	}
}

// IdentityFormatter returns a no-op formatter pair.
func IdentityFormatter() (inputFormatter func([]byte) []byte, outputFormatter func([]byte) []byte) {
	f := func(inpData []byte) []byte { return inpData }
	return f, f
}

// FormatterForRelayRequestAndResponseJsonRPC returns formatters that remove the JSON-RPC id from
// requests (so cache keys are id-independent) and restore the original id in responses.
func FormatterForRelayRequestAndResponseJsonRPC() (inputFormatter func([]byte) []byte, outputFormatter func([]byte) []byte) {
	var extractedID interface{} = "-1"
	extractedIDArray := []interface{}{}

	inputFormatter = func(inpData []byte) []byte {
		if len(inpData) < 3 {
			return inpData
		}
		batch := []json.RawMessage{}
		err := json.Unmarshal(inpData, &batch)
		if err == nil && len(batch) >= 1 {
			modifiedInpArray := []json.RawMessage{}
			for _, batchData := range batch {
				var extractedIDForBatch interface{}
				var modifiedInp []byte
				modifiedInp, extractedIDForBatch, err = getExtractedIDAndModifyInputForJSON(batchData)
				if err != nil {
					return inpData
				}
				extractedIDArray = append(extractedIDArray, extractedIDForBatch)
				modifiedInpArray = append(modifiedInpArray, modifiedInp)
			}
			modifiedOut, err := json.Marshal(modifiedInpArray)
			if err != nil {
				utils.LavaFormatError("failed to marshal batch", err)
				return inpData
			}
			return modifiedOut
		}
		var modifiedInp []byte
		modifiedInp, extractedID, err = getExtractedIDAndModifyInputForJSON(inpData)
		if err != nil {
			return inpData
		}
		return modifiedInp
	}

	outputFormatter = func(inpData []byte) []byte {
		if len(inpData) == 0 {
			return inpData
		}
		batch := []json.RawMessage{}
		err := json.Unmarshal(inpData, &batch)
		if err == nil && len(batch) >= 1 && len(extractedIDArray) == len(batch) {
			modifiedInpArray := []json.RawMessage{}
			for i, batchData := range batch {
				modifiedInp, err := sjson.SetBytes(batchData, IDFieldName, extractedIDArray[i])
				if err != nil {
					utils.LavaFormatWarning("failed to set id in batch cache", err)
					return inpData
				}
				modifiedInpArray = append(modifiedInpArray, modifiedInp)
			}
			modifiedOut, err := json.Marshal(modifiedInpArray)
			if err != nil {
				utils.LavaFormatError("failed to marshal batch", err)
				return inpData
			}
			return modifiedOut
		}
		modifiedInp, err := sjson.SetBytes(inpData, IDFieldName, extractedID)
		if err != nil {
			utils.LavaFormatWarning("failed to set input id in cache", err)
			return inpData
		}
		return modifiedInp
	}

	return inputFormatter, outputFormatter
}

func getExtractedIDAndModifyInputForJSON(inpData []byte) (modifiedInp []byte, extractedID interface{}, err error) {
	result := gjson.GetBytes(inpData, IDFieldName)
	switch result.Type {
	case gjson.Number:
		extractedID = result.Int()
	case gjson.String:
		extractedID = result.Raw
	default:
		extractedID = result.Value()
	}
	modifiedInp, err = sjson.SetBytes(inpData, IDFieldName, DefaultIDValue)
	if err != nil {
		return inpData, extractedID, utils.LavaFormatWarning("failed to set id in json", err,
			utils.Attribute{Key: "jsonData", Value: inpData},
			utils.LogAttr("extractedID", extractedID),
		)
	}
	return modifiedInp, extractedID, nil
}
