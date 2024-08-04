package extensionslib

import (
	spectypes "github.com/lavanet/lava/v2/x/spec/types"
)

type ArchiveParserRule struct {
	extension *spectypes.Extension
}

func (apr ArchiveParserRule) isPassingRule(extensionChainMessage ExtensionsChainMessage, latestBlock uint64) bool {
	_, earliestRequestedBlock := extensionChainMessage.RequestedBlock()
	if earliestRequestedBlock < 0 {
		// if asking for the latest block, or an api that doesn't have a specific block requested then it's not archive
		return earliestRequestedBlock == spectypes.EARLIEST_BLOCK // only earliest should go to archive
	}
	if latestBlock == 0 {
		return true
	}
	if uint64(earliestRequestedBlock) >= latestBlock {
		return false
	}
	if apr.extension.Rule != nil && apr.extension.Rule.Block != 0 {
		if latestBlock-apr.extension.Rule.Block > uint64(earliestRequestedBlock) {
			return true
		}
	}
	return false
}
