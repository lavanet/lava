package extensionslib

type ArchiveParserRule struct {
}

func (arp ArchiveParserRule) isPassingRule(extensionChainMessage ExtensionsChainMessage, latestBlock uint64) bool {
	return true
}
