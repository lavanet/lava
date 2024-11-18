package extensionslib

import (
	spectypes "github.com/lavanet/lava/v4/x/spec/types"
)

const ArchiveExtension = "archive"

type ExtensionInfo struct {
	ExtensionOverride    []string
	LatestBlock          uint64
	AdditionalExtensions []string
}

type ExtensionsChainMessage interface {
	SetExtension(*spectypes.Extension)
	RequestedBlock() (latest int64, earliest int64)
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
	allowedExtensions    map[string]struct{}
	configuredExtensions map[ExtensionKey]*spectypes.Extension
}

func NewExtensionParser(allowedExtensions map[string]struct{}, configuredExtensions map[ExtensionKey]*spectypes.Extension) ExtensionParser {
	return ExtensionParser{
		allowedExtensions:    allowedExtensions,
		configuredExtensions: configuredExtensions,
	}
}

func (ep *ExtensionParser) GetExtension(extension ExtensionKey) *spectypes.Extension {
	if extension.Extension == "" {
		return nil
	}
	extensionRet, ok := ep.configuredExtensions[extension]
	if ok {
		return extensionRet
	}
	return nil
}

func (ep *ExtensionParser) GetConfiguredExtensions() map[ExtensionKey]*spectypes.Extension {
	return ep.configuredExtensions
}

func (ep *ExtensionParser) AllowedExtension(extension string) bool {
	if extension == "" {
		return true
	}
	_, ok := ep.allowedExtensions[extension]
	return ok
}

func (ep *ExtensionParser) SetConfiguredExtensions(configuredExtensions map[ExtensionKey]*spectypes.Extension) {
	ep.configuredExtensions = configuredExtensions
}

func (ep *ExtensionParser) ExtensionParsing(addon string, extensionsChainMessage ExtensionsChainMessage, latestBlock uint64) {
	if len(ep.configuredExtensions) == 0 {
		return
	}
	for extensionKey, extension := range ep.configuredExtensions {
		if extensionKey.Addon != addon {
			// this extension is not relevant for this api
			continue
		}
		extensionParserRule := NewExtensionParserRule(extension)
		if extensionParserRule != nil && extensionParserRule.isPassingRule(extensionsChainMessage, latestBlock) {
			extensionsChainMessage.SetExtension(extension)
		}
	}
}

func NewExtensionParserRule(extension *spectypes.Extension) ExtensionParserRule {
	switch extension.Name {
	case ArchiveExtension:
		return ArchiveParserRule{extension: extension}
	default:
		// unsupported rule
		return nil
	}
}
