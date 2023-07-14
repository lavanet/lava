package upgrade

import (
	"encoding/json"
	"strconv"
	"strings"
	"sync"

	"github.com/lavanet/lava/utils"
	protocoltypes "github.com/lavanet/lava/x/protocol/types"
	terderminttypes "github.com/tendermint/tendermint/abci/types"
)

type ProtocolVersion struct {
	ProviderTarget string
	ProviderMin    string
	ConsumerTarget string
	ConsumerMin    string
}

var LavaProtocolVersion = ProtocolVersion{
	ProviderTarget: "0.16.0",
	ProviderMin:    "0.16.0",
	ConsumerTarget: "0.16.0",
	ConsumerMin:    "0.16.0",
}

type UpgradeManager struct {
	lock sync.RWMutex
}

// Returning a new provider session manager
func NewUpdateManager() *UpgradeManager {
	return &UpgradeManager{}
}

func (um *UpgradeManager) SetProtocolVersion(newVersion *protocoltypes.Version) {
	um.lock.Lock()
	defer um.lock.Unlock()
	LavaProtocolVersion.ProviderMin = newVersion.ProviderMin
	LavaProtocolVersion.ProviderTarget = newVersion.ProviderTarget
	LavaProtocolVersion.ConsumerTarget = newVersion.ConsumerTarget
	LavaProtocolVersion.ConsumerMin = newVersion.ConsumerMin
}

func BuildVersionFromParamChangeEvent(event terderminttypes.Event) bool {
	foundVersionParam := false
	var versionValue []byte

	for _, attribute := range event.Attributes {
		key := string(attribute.Key)
		value := string(attribute.Value)

		if key == "param" {
			if value == "Version" {
				foundVersionParam = true
			} else {
				// Reset the flag if we encounter a "param" not equal to "Version"
				foundVersionParam = false
			}
		} else if key == "value" && foundVersionParam {
			versionValue = attribute.Value
			break
		}
	}
	// if versionValue not found, return false
	if versionValue == nil {
		return false
	}
	var version *ProtocolVersion
	// We are making sure that proposal value can be unmarshalled with ProtocolVersion type
	err := json.Unmarshal(versionValue, &version)
	return err == nil
}

// helper function to parse version major/middle/minor fields
type ParsedVersion struct {
	Major  int
	Middle int
	Minor  int
}

func ParseVersion(versionString string) (ParsedVersion, error) {
	splitVersion := strings.Split(versionString, ".")
	if len(splitVersion) != 3 {
		return ParsedVersion{}, utils.LavaFormatError("invalid version string", nil)
	}

	major, err := strconv.Atoi(splitVersion[0])
	if err != nil {
		return ParsedVersion{}, err
	}

	middle, err := strconv.Atoi(splitVersion[1])
	if err != nil {
		return ParsedVersion{}, err
	}

	minor, err := strconv.Atoi(splitVersion[2])
	if err != nil {
		return ParsedVersion{}, err
	}

	return ParsedVersion{Major: major, Middle: middle, Minor: minor}, nil
}

func ParseMultipleVersions(versions []string) ([]ParsedVersion, error) {
	parsedVersions := make([]ParsedVersion, len(versions))

	for i, version := range versions {
		parsed, err := ParseVersion(version)
		if err != nil {
			return nil, err
		}

		parsedVersions[i] = parsed
	}

	return parsedVersions, nil
}

func ParseLavadVersion(version string) (ParsedVersion, error) {
	parts := strings.Split(version, "-")
	if len(parts) == 0 {
		return ParsedVersion{}, utils.LavaFormatError("invalid version format", nil)
	}
	return ParseVersion(parts[0])
}
