package extensionslib

import (
	spectypes "github.com/lavanet/lava/x/spec/types"
)

type ExtensionsChainMessage interface {
	SetExtension(*spectypes.Extension)
	RequestedBlock() int64
}

type ExtensionKey struct {
	Extension      string
	ConnectionType string
	InternalPath   string
	Addon          string
}

type ExtensionParserRule interface {
	isPassingRule(extensionChainMessage ExtensionsChainMessage, latestBlock uint64) bool
}

type ExtensionParser struct {
	AllowedExtensions    map[string]struct{}
	configuredExtensions map[ExtensionKey]*spectypes.Extension
}

func (ep *ExtensionParser) AllowedExtension(extension string) bool {
	_, ok := ep.AllowedExtensions[extension]
	return ok
}

func (ep *ExtensionParser) SetConfiguredExtensions(configuredExtensions map[ExtensionKey]*spectypes.Extension) {
	ep.configuredExtensions = configuredExtensions
}

func (ep *ExtensionParser) ExtensionParsing(extensionsChainMessage ExtensionsChainMessage, latestBlock uint64) {
	if len(ep.configuredExtensions) == 0 {
		return
	}

	for _, extension := range ep.configuredExtensions {
		extensionParserRule := NewExtensionParserRule(extension)
		if extensionParserRule.isPassingRule(extensionsChainMessage, latestBlock) {
			extensionsChainMessage.SetExtension(extension)
		}
	}
}

func NewExtensionParserRule(extension *spectypes.Extension) ExtensionParserRule {
	switch extension.Name {
	case "archive":
		return ArchiveParserRule{extension: extension}
	default:
		// unsupported rule
		return nil
	}
}
