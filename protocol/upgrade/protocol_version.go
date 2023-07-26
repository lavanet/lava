package upgrade

import (
	"strconv"
	"strings"

	"github.com/lavanet/lava/utils"
	protocoltypes "github.com/lavanet/lava/x/protocol/types"
)

var LavaProtocolVersion = protocoltypes.Version{
	ProviderTarget: "0.21.0",
	ProviderMin:    "0.21.0",
	ConsumerTarget: "0.21.0",
	ConsumerMin:    "0.21.0",
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
