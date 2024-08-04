package common

import (
	"encoding/hex"
	"regexp"
	"strconv"

	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/lavanet/lava/v2/utils"
	"github.com/spf13/pflag"
)

// extract requested sequence number from tx error.
func FindSequenceNumber(sequence string) (int, error) {
	re := regexp.MustCompile(`expected (\d+), got (\d+)`)
	match := re.FindStringSubmatch(sequence)
	if match == nil || len(match) < 2 {
		return 0, utils.LavaFormatWarning("Failed to parse sequence number from error", nil, utils.Attribute{Key: "sequence", Value: sequence})
	}
	return strconv.Atoi(match[1]) // atoi return 0 upon error, so it will be ok when sequenceNumberParsed uses it
}

type TxResultData struct {
	RawLog string
	Txhash []byte
	Code   int
}

func ParseTransactionResult(parsedValues map[string]any) (retData TxResultData, err error) {
	ret := TxResultData{}
	utils.LavaFormatDebug("result", utils.LogAttr("parsedValues", parsedValues))
	txHash, ok := parsedValues["txhash"]
	if ok {
		txHashStr, ok := txHash.(string)
		if ok {
			ret.Txhash, err = hex.DecodeString(txHashStr)
		} else {
			err = utils.LavaFormatError("failed parsing txHash", nil, utils.Attribute{Key: "parsedValues", Value: parsedValues})
		}
	} else {
		err = utils.LavaFormatWarning("failed parsing txHash", nil, utils.Attribute{Key: "parsedValues", Value: parsedValues})
	}
	rawlog, ok := parsedValues["raw_log"]
	if ok {
		rawLogStr, ok := rawlog.(string)
		if ok {
			ret.RawLog = rawLogStr
		} else {
			err = utils.LavaFormatError("failed parsing rawlog", nil, utils.Attribute{Key: "parsedValues", Value: parsedValues})
		}
	}
	code, ok := parsedValues["code"]
	if ok {
		codeNumber, ok := code.(float64)
		if ok {
			ret.Code = int(codeNumber)
		} else {
			err = utils.LavaFormatError("failed parsing code", nil, utils.Attribute{Key: "parsedValues", Value: parsedValues})
		}
	}
	return ret, err
}

func VerifyAndHandleUnsupportedFlags(currentFlags *pflag.FlagSet) error {
	fees, err := currentFlags.GetString(flags.FlagFees)
	if err != nil {
		return err
	}
	if fees != "" {
		currentFlags.Set(flags.FlagFees, "")
		utils.LavaFormatWarning("fees flag was used and is not supported, rpcprovider will ignore it. No action is required", nil)
	}
	return nil
}
