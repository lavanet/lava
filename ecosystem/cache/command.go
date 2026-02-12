package cache

import (
	"context"

	"github.com/lavanet/lava/v5/protocol/performance"
	"github.com/lavanet/lava/v5/utils"
	"github.com/rs/zerolog"
	"github.com/spf13/cobra"
)

const (
	FlagLogLevel       = "log_level"
	FlagMetricsAddress = "metrics_address"
)

func CreateCacheCobraCommand() *cobra.Command {
	cacheCmd := &cobra.Command{
		Use:   "cache [address<HOST:PORT> | unix:<SOCKET_PATH.sock>]",
		Short: "set up a ram based cache server for relays listening on address specified, can work either with rpcconsumer or rpcprovider to improve latency",
		Long: `set up a ram based cache server for relays listening on address specified, can work either with rpcconsumer or rpcprovider to improve latency,
longer DefaultExpirationForNonFinalized will reduce sync QoS for "latest" request but reduce load`,
		Example: `cache "127.0.0.1:7777"`,
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			address := args[0]

			ctx := context.Background()
			logLevel, err := cmd.Flags().GetString(FlagLogLevel)
			if err != nil {
				utils.LavaFormatFatal("failed to read log level flag", err)
			}
			utils.SetGlobalLoggingLevel(logLevel)

			// check if the command includes --pyroscope-address
			pyroscopeAddressFlagUsed := cmd.Flags().Lookup(performance.PyroscopeAddressFlagName).Changed
			if pyroscopeAddressFlagUsed {
				pyroscopeServerAddress, err := cmd.Flags().GetString(performance.PyroscopeAddressFlagName)
				if err != nil {
					utils.LavaFormatFatal("failed to read pyroscope address flag", err)
				}
				pyroscopeAppName, err := cmd.Flags().GetString(performance.PyroscopeAppNameFlagName)
				if err != nil || pyroscopeAppName == "" {
					pyroscopeAppName = "lavap-cache"
				}
				mutexProfileFraction, err := cmd.Flags().GetInt(performance.PyroscopeMutexProfileFractionFlagName)
				if err != nil {
					mutexProfileFraction = performance.DefaultMutexProfileFraction
				}
				blockProfileRate, err := cmd.Flags().GetInt(performance.PyroscopeBlockProfileRateFlagName)
				if err != nil {
					blockProfileRate = performance.DefaultBlockProfileRate
				}
				tagsStr, _ := cmd.Flags().GetString(performance.PyroscopeTagsFlagName)
				tags := performance.ParseTags(tagsStr)
				err = performance.StartPyroscope(pyroscopeAppName, pyroscopeServerAddress, mutexProfileFraction, blockProfileRate, tags)
				if err != nil {
					return utils.LavaFormatError("failed to start pyroscope profiler", err)
				}
			}

			metricsAddress, err := cmd.Flags().GetString(FlagMetricsAddress)
			if err != nil {
				utils.LavaFormatFatal("failed to read metrics address flag", err)
			}
			Server(ctx, address, metricsAddress, cmd.Flags())
			return nil
		},
	}
	cacheCmd.Flags().String(FlagLogLevel, zerolog.InfoLevel.String(), "The logging level (trace|debug|info|warn|error|fatal|panic)")
	cacheCmd.Flags().Duration(ExpirationFlagName, DefaultExpirationTimeFinalized, "how long does a cache entry lasts in the cache for a finalized entry")
	cacheCmd.Flags().Duration(ExpirationNonFinalizedFlagName, DefaultExpirationForNonFinalized, "how long does a cache entry lasts in the cache for a non finalized entry")
	cacheCmd.Flags().Float64(ExpirationTimeFinalizedMultiplierFlagName, DefaultExpirationTimeFinalizedMultiplier, "Multiplier for finalized cache entry expiration. 1 means no change (default), 1.2 means 20% longer.")
	cacheCmd.Flags().Float64(ExpirationTimeNonFinalizedMultiplierFlagName, DefaultExpirationTimeNonFinalizedMultiplier, "Multiplier for non-finalized cache entry expiration. 1 means no change (default), 1.2 means 20% longer.")
	cacheCmd.Flags().Duration(ExpirationBlocksHashesToHeightsFlagName, DefaultExpirationBlocksHashesToHeights, "how long does the cache entry lasts in the cache for a block hash to height entry")
	cacheCmd.Flags().Duration(ExpirationNodeErrorsOnFinalizedFlagName, DefaultExpirationNodeErrors, "how long does a cache entry lasts in the cache for a finalized node error entry")
	cacheCmd.Flags().String(FlagMetricsAddress, DisabledFlagOption, "address to listen to prometheus metrics 127.0.0.1:5555, later you can curl http://127.0.0.1:5555/metrics")
	cacheCmd.Flags().Int64(FlagCacheSizeName, 2*1024*1024*1024, "the maximal amount of entries to save")
	cacheCmd.Flags().String(performance.PyroscopeAddressFlagName, "", "pyroscope server address for continuous profiling (e.g., http://pyroscope:4040)")
	cacheCmd.Flags().String(performance.PyroscopeAppNameFlagName, "lavap-cache", "pyroscope application name for identifying this service")
	cacheCmd.Flags().Int(performance.PyroscopeMutexProfileFractionFlagName, performance.DefaultMutexProfileFraction, "mutex profile sampling rate (1 in N mutex events)")
	cacheCmd.Flags().Int(performance.PyroscopeBlockProfileRateFlagName, performance.DefaultBlockProfileRate, "block profile rate in nanoseconds (1 records all blocking events)")
	cacheCmd.Flags().String(performance.PyroscopeTagsFlagName, "", "comma-separated list of tags in key=value format (e.g., instance=cache-1,region=us-east)")
	return cacheCmd
}
