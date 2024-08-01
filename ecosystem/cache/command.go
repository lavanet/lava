package cache

import (
	"context"

	"github.com/lavanet/lava/v2/utils"
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
	cacheCmd.Flags().Duration(ExpirationNodeErrorsOnFinalizedFlagName, DefaultExpirationNodeErrors, "how long does a cache entry lasts in the cache for a finalized node error entry")
	cacheCmd.Flags().String(FlagMetricsAddress, DisabledFlagOption, "address to listen to prometheus metrics 127.0.0.1:5555, later you can curl http://127.0.0.1:5555/metrics")
	cacheCmd.Flags().Int64(FlagCacheSizeName, 2*1024*1024*1024, "the maximal amount of entries to save")
	return cacheCmd
}
