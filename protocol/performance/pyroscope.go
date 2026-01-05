package performance

import (
	"github.com/grafana/pyroscope-go"
	"github.com/lavanet/lava/v5/utils"
)

const (
	// PyroscopeAddressFlagName is the flag name for the Pyroscope server address
	PyroscopeAddressFlagName = "pyroscope-address"
	// PyroscopeAppNameFlagName is the flag name for the Pyroscope application name
	PyroscopeAppNameFlagName = "pyroscope-app-name"
)

// StartPyroscope initializes continuous profiling with Pyroscope
// It connects to a Pyroscope server and sends profiling data for:
// - CPU profiling
// - Memory allocation (objects and bytes)
// - Memory in-use (objects and bytes)
// - Goroutine profiling
func StartPyroscope(appName, serverAddress string) error {
	_, err := pyroscope.Start(pyroscope.Config{
		ApplicationName: appName,
		ServerAddress:   serverAddress,
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
