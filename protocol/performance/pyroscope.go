package performance

import (
	"runtime"
	"strings"

	"github.com/grafana/pyroscope-go"
	"github.com/lavanet/lava/v5/utils"
)

const (
	// PyroscopeAddressFlagName is the flag name for the Pyroscope server address
	PyroscopeAddressFlagName = "pyroscope-address"
	// PyroscopeAppNameFlagName is the flag name for the Pyroscope application name
	PyroscopeAppNameFlagName = "pyroscope-app-name"
	// PyroscopeMutexProfileFractionFlagName is the flag name for the mutex profile sampling rate
	PyroscopeMutexProfileFractionFlagName = "pyroscope-mutex-profile-fraction"
	// PyroscopeBlockProfileRateFlagName is the flag name for the block profile rate
	PyroscopeBlockProfileRateFlagName = "pyroscope-block-profile-rate"
	// PyroscopeTagsFlagName is the flag name for custom tags to add to profiles
	PyroscopeTagsFlagName = "pyroscope-tags"

	// DefaultMutexProfileFraction is the default sampling rate for mutex profiling (1 in N events)
	DefaultMutexProfileFraction = 5
	// DefaultBlockProfileRate is the default rate for block profiling in nanoseconds
	DefaultBlockProfileRate = 1
)

// ParseTags parses a comma-separated list of key=value pairs into a map.
// Example input: "instance=provider-1,region=us-east,chain=ETH1"
func ParseTags(tagsStr string) map[string]string {
	tags := make(map[string]string)
	if tagsStr == "" {
		return tags
	}
	pairs := strings.Split(tagsStr, ",")
	for _, pair := range pairs {
		kv := strings.SplitN(strings.TrimSpace(pair), "=", 2)
		if len(kv) == 2 {
			tags[strings.TrimSpace(kv[0])] = strings.TrimSpace(kv[1])
		}
	}
	return tags
}

// StartPyroscope initializes continuous profiling with Pyroscope
// It connects to a Pyroscope server and sends profiling data for:
// - CPU profiling
// - Memory allocation (objects and bytes)
// - Memory in-use (objects and bytes)
// - Goroutine profiling
func StartPyroscope(appName, serverAddress string, mutexProfileFraction, blockProfileRate int, tags map[string]string) error {
	// Enable mutex and block profiling in Go runtime
	runtime.SetMutexProfileFraction(mutexProfileFraction) // Sample 1 in N mutex events
	runtime.SetBlockProfileRate(blockProfileRate)         // Record blocking events (in nanoseconds)

	_, err := pyroscope.Start(pyroscope.Config{
		ApplicationName: appName,
		ServerAddress:   serverAddress,
		Tags:            tags,
		ProfileTypes: []pyroscope.ProfileType{
			pyroscope.ProfileCPU,
			pyroscope.ProfileAllocObjects,
			pyroscope.ProfileAllocSpace,
			pyroscope.ProfileInuseObjects,
			pyroscope.ProfileInuseSpace,
			pyroscope.ProfileGoroutines,
			pyroscope.ProfileMutexCount,
			pyroscope.ProfileMutexDuration,
			pyroscope.ProfileBlockCount,
			pyroscope.ProfileBlockDuration,
		},
	})
	if err != nil {
		return utils.LavaFormatError("failed to start pyroscope profiler", err)
	}

	utils.LavaFormatInfo("started pyroscope profiler",
		utils.Attribute{Key: "application", Value: appName},
		utils.Attribute{Key: "server", Value: serverAddress},
	)

	return nil
}
