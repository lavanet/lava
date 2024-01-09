package extensionslib

import (
	"github.com/lavanet/lava/utils"
	spectypes "github.com/lavanet/lava/x/spec/types"
)

type ArchiveParserRule struct {
	extension *spectypes.Extension
}

func (apr ArchiveParserRule) isPassingRule(extensionChainMessage ExtensionsChainMessage, latestBlock uint64) bool {
	_, earliestRequestedBlock := extensionChainMessage.RequestedBlock()
	utils.LavaFormatDebug("isPassingRule 1",
		utils.LogAttr("earliestRequestedBlock", earliestRequestedBlock),
	)
	if earliestRequestedBlock < 0 {
		// if asking for the latest block, or an api that doesn't have a specific block requested then it's not archive
		return earliestRequestedBlock == spectypes.EARLIEST_BLOCK // only earliest should go to archive
	}
	utils.LavaFormatDebug("isPassingRule 2",
		utils.LogAttr("latestBlock", latestBlock),
	)
	if latestBlock == 0 {
		return true
	}
	if uint64(earliestRequestedBlock) >= latestBlock {
		return false
	}

	if apr.extension.Rule != nil && apr.extension.Rule.Block != 0 {
		if latestBlock-apr.extension.Rule.Block > uint64(earliestRequestedBlock) {
			utils.LavaFormatDebug("isPassingRule 3",
				utils.LogAttr("latestBlock-apr.extension.Rule.Block", latestBlock-apr.extension.Rule.Block),
				utils.LogAttr("earliestRequestedBlock", earliestRequestedBlock),
			)
			return true
		}
	}
	return false
}
