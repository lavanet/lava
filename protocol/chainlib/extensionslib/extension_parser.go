package extensionslib

import (
	"github.com/lavanet/lava/utils"
	spectypes "github.com/lavanet/lava/x/spec/types"
)

// struct used for message parsing to determine if extension override is required or not
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
	utils.LavaFormatDebug("Extension parsing",
		utils.LogAttr("addon", addon),
	)

	for extensionKey, extension := range ep.configuredExtensions {
		if extensionKey.Addon != addon {
			// this extension is not relevant for this api
			continue
		}
		utils.LavaFormatDebug("Extension parsing didnt continue.",
			utils.LogAttr("extensionKey", extensionKey),
		)
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
