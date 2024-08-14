package extensionslib

import (
	"github.com/lavanet/lava/v2/utils/maps"
	spectypes "github.com/lavanet/lava/v2/x/spec/types"
)

const (
	ExtensionTypeArchive = "archive"
)

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
	AllowedExtensions    map[string]struct{}
	configuredExtensions map[ExtensionKey]*spectypes.Extension
}

func (ep *ExtensionParser) GetExtensionByName(extensionName string) *spectypes.Extension {
	if extensionName == "" {
		return nil
	}

	findExtensionsPredicate := func(key ExtensionKey, _ *spectypes.Extension) bool {
		return key.Extension == extensionName
	}

	_, archiveExt, found := maps.FindInMap(ep.configuredExtensions, findExtensionsPredicate)
	if found {
		return archiveExt
	}

	return nil
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
	_, ok := ep.AllowedExtensions[extension]
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
		if extensionParserRule.isPassingRule(extensionsChainMessage, latestBlock) {
			extensionsChainMessage.SetExtension(extension)
		}
	}
}

func NewExtensionParserRule(extension *spectypes.Extension) ExtensionParserRule {
	switch extension.Name {
	case ExtensionTypeArchive:
		return ArchiveParserRule{extension: extension}
	default:
		// unsupported rule
		return nil
	}
}

// this wrapper is used to return a different earliest
type EarliestOverriddenExtensionChainMessage struct {
	earliest             int64
	setExtensionCallback func(extension *spectypes.Extension)
}

func NewEarliestOverriddenExtensionChainMessage(earliest int64, setExtensionCallback func(extension *spectypes.Extension)) *EarliestOverriddenExtensionChainMessage {
	return &EarliestOverriddenExtensionChainMessage{
		earliest:             earliest,
		setExtensionCallback: setExtensionCallback,
	}
}

func (cm *EarliestOverriddenExtensionChainMessage) SetExtension(extension *spectypes.Extension) {
	cm.setExtensionCallback(extension)
}

func (cm *EarliestOverriddenExtensionChainMessage) RequestedBlock() (latest int64, earliest int64) {
	// The latest is irrelevant, we only care about the earliest
	return spectypes.LATEST_BLOCK, cm.earliest
}
