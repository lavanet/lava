package extensionslib

import spectypes "github.com/lavanet/lava/x/spec/types"

type ArchiveParserRule struct {
	extension *spectypes.Extension
}

func (arp ArchiveParserRule) isPassingRule(extensionChainMessage ExtensionsChainMessage, latestBlock uint64) bool {
	requestedBlock := extensionChainMessage.RequestedBlock()
	if requestedBlock < 0 {
		// if asking for the latest block, or an api that doesn't have a specific block requested then it's not archive
		return false
	}
	if latestBlock == 0 {
		return true
	}
	if uint64(requestedBlock) >= latestBlock {
		return false
	}
	if arp.extension.Rule != nil && arp.extension.Rule.Block != 0 {
		if latestBlock-arp.extension.Rule.Block > uint64(requestedBlock) {
			return true
		}
	}
	return false
}
