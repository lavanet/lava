package common

import (
	"regexp"
	"strconv"
	"strings"

	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/lavanet/lava/utils"
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

func ParseTransactionResult(transactionResult string) (string, int) {
	transactionResult = strings.ReplaceAll(transactionResult, ": ", ":")
	transactionResults := strings.Split(transactionResult, "\n")
	summarizedResult := ""
	for _, str := range transactionResults {
		if strings.Contains(str, "raw_log:") || strings.Contains(str, "txhash:") || strings.Contains(str, "code:") {
			summarizedResult = summarizedResult + str + ", "
		}
	}

	re := regexp.MustCompile(`code:(\d+)`) // extracting code from transaction result (in format code:%d)
	match := re.FindStringSubmatch(transactionResult)
	if match == nil || len(match) < 2 {
		return summarizedResult, 1 // not zero
	}
	retCode, err := strconv.Atoi(match[1]) // extract return code.
	if err != nil {
		return summarizedResult, 1 // not zero
	}
	return summarizedResult, retCode
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
