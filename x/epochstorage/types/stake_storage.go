package types

import (
	fmt "fmt"
	"strings"

	"cosmossdk.io/math"
	"github.com/lavanet/lava/utils"
)

var (
	StakeEntriesPrefix        = []byte("StakeEntries/")
	StakeEntriesCurrentPrefix = []byte("StakeEntriesCurrent/")
)

func StakeEntryKey(epoch uint64, chainID string, stake math.Int, provider string) []byte {
	key := append(utils.Serialize(epoch), []byte(" "+chainID+" ")...)
	key = append(key, utils.Serialize(stake.Uint64())...)
	key = append(key, []byte(" "+provider)...)
	return key
}

func StakeEntryKeyCurrent(chainID string, provider string) []byte {
	return []byte(strings.Join([]string{chainID, provider}, " "))
}

func ExtractEpochFromStakeEntryKey(key string) (epoch uint64, err error) {
	if len(key) < 8 {
		return 0, fmt.Errorf("ExtractEpochFromStakeEntryKey: invalid StakeEntryKey, bad structure. key: %s", key)
	}

	utils.Deserialize([]byte(key[:8]), &epoch)
	return epoch, nil
}
