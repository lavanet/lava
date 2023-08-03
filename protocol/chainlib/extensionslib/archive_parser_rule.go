package extensionslib

import spectypes "github.com/lavanet/lava/x/spec/types"

type ArchiveParserRule struct {
	extension *spectypes.Extension
}

func (arp ArchiveParserRule) isPassingRule(extensionChainMessage ExtensionsChainMessage, latestBlock uint64) bool {
	requestedBlock := extensionChainMessage.RequestedBlock()
	if latestBlock == 0 {
		return true
	}
	if requestedBlock < 0 || uint64(requestedBlock) >= latestBlock {
		return false
	}
	if arp.extension.Rule != nil && arp.extension.Rule.Block != 0 {
		if latestBlock-arp.extension.Rule.Block > uint64(requestedBlock) {
			return true
		}
	}
	return false
}
