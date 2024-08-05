package types

import (
	fmt "fmt"

	"github.com/lavanet/lava/v2/utils"
)

func (ss StakeStorage) GetStakeEntryByAddressFromStorage(address string) (StakeEntry, bool) {
	if !utils.IsBech32Address(address) {
		utils.LavaFormatWarning("address is not Bech32", fmt.Errorf("invalid address"),
			utils.LogAttr("address", address),
		)
		return StakeEntry{}, false
	}

	for _, entry := range ss.StakeEntries {
		if !utils.IsBech32Address(entry.Address) || !utils.IsBech32Address(entry.Vault) {
			// this should not happen; to avoid panic we simply skip this one (thus
			// freeze the situation so it can be investigated and orderly resolved).
			utils.LavaFormatError("critical: invalid account address inside StakeStorage", fmt.Errorf("invalid address"),
				utils.LogAttr("provider", entry.Address),
				utils.LogAttr("vault", entry.Vault),
				utils.LogAttr("chainID", entry.Chain),
			)
			continue
		}

		if entry.IsAddressVaultOrProvider(address) {
			// found the right entry
			return entry, true
		}
	}

	return StakeEntry{}, false
}
