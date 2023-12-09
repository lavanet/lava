package monitoring

import (
	"context"
	"os"
	"os/signal"
	"time"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/config"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/lavanet/lava/app"
	"github.com/lavanet/lava/protocol/common"
	"github.com/lavanet/lava/protocol/lavasession"
	"github.com/lavanet/lava/protocol/metrics"
	"github.com/lavanet/lava/utils"
	"github.com/lavanet/lava/utils/rand"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

const (
	ProviderAddressesPropertyName     = "provider_addresses"
	SubscriptionAddressesPropertyName = "subscription_addresses"
	intervalFlagName                  = "interval"
	intervalDefaultDuration           = 0 * time.Second
	ConsumerEndpointPropertyName      = "consumer_endpoints"
	ReferenceEndpointPropertyName     = "reference_endpoints"
	allowedBlockTimeLagFlagName       = "allowed_time_lag"
	QueryRetriesFlagName              = "query-retries"
	RunLabelFlagName                  = "run-label"
	allowedBlockTimeDefaultLag        = 30 * time.Second
)

func ParseEndpoints(keyName string, viper_endpoints *viper.Viper) (endpoints []*lavasession.RPCEndpoint, err error) {
	err = viper_endpoints.UnmarshalKey(keyName, &endpoints)
	if err != nil {
		utils.LavaFormatError("could not unmarshal key to endpoints", err, utils.LogAttr("key", keyName), utils.Attribute{Key: "viper_endpoints", Value: viper_endpoints.AllSettings()})
	}
	return
}

func CreateTestHealthCobraCommand() *cobra.Command {
	cmdTestHealth := &cobra.Command{
		Use:   `health config_file`,
		Short: `start monitoring the health of the protocol processes defined in the config`,
		Long:  `config_file if a path to a yml file`,
		Example: `health config/health_examples/health_config.yml
example health config files can be found in https://github.com/lavanet/lava/blob/main/config/health_examples
subscription_addresses:
	- lava@...
	- lava@...
provider_addresses:
	- lava@...
	- lava@...
	- lava@...
consumer_endpoints:
    - chain-id: ETH1
      api-interface: jsonrpc
      network-address: 127.0.0.1:3333
reference_endpoints:
	- chain-id: ETH1
      api-interface: jsonrpc
      network-address: public-rpc-1
	- chain-id: ETH1
      api-interface: jsonrpc
      network-address: public-rpc-2`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}
			ctx, cancel := context.WithCancel(context.Background())
			signalChan := make(chan os.Signal, 1)
			signal.Notify(signalChan, os.Interrupt)
			defer func() {
				signal.Stop(signalChan)
				cancel()
			}()
			config_name := args[0] // name of config file (without extension)
			viper.SetConfigName(config_name)
			viper.SetConfigType("yml")
			viper.AddConfigPath(".")
			viper.AddConfigPath("./config")
			viper.AddConfigPath(app.DefaultNodeHome)
			err = viper.ReadInConfig()
			if err != nil {
				utils.LavaFormatFatal("could not load config file", err, utils.Attribute{Key: "expected_config_name", Value: viper.ConfigFileUsed()})
			}
			// set log format
			logFormat := viper.GetString(flags.FlagLogFormat)
			utils.JsonFormat = logFormat == "json"
			// set rolling log.
			closeLoggerOnFinish := common.SetupRollingLogger()
			defer closeLoggerOnFinish()
			logLevel, err := cmd.Flags().GetString(flags.FlagLogLevel)
			if err != nil {
				utils.LavaFormatFatal("failed to read log level flag", err)
			}
			utils.SetGlobalLoggingLevel(logLevel)
			networkChainId := viper.GetString(flags.FlagChainID)
			if networkChainId == app.Name {
				clientTomlConfig, err := config.ReadFromClientConfig(clientCtx)
				if err == nil {
					if clientTomlConfig.ChainID != "" {
						networkChainId = clientTomlConfig.ChainID
					}
				}
			}
			clientCtx = clientCtx.WithChainID(networkChainId)
			rand.InitRandomSeed()
			runLabel := viper.GetString(RunLabelFlagName)
			prometheusListenAddr := viper.GetString(metrics.MetricsListenFlagName)
			providerAddresses := viper.GetStringSlice(ProviderAddressesPropertyName)
			subscriptionAddresses := viper.GetStringSlice(SubscriptionAddressesPropertyName)
			keyName := ConsumerEndpointPropertyName
			consumerEndpoints, _ := ParseEndpoints(keyName, viper.GetViper())
			keyName = ReferenceEndpointPropertyName
			referenceEndpoints, _ := ParseEndpoints(keyName, viper.GetViper())
			interval := viper.GetDuration(intervalFlagName)
			healthMetrics := metrics.NewHealthMetrics(prometheusListenAddr)
			RunHealthCheck := func(ctx context.Context,
				clientCtx client.Context,
				subscriptionAddresses []string,
				providerAddresses []string,
				consumerEndpoints []*lavasession.RPCEndpoint,
				referenceEndpoints []*lavasession.RPCEndpoint,
				prometheusListenAddr string) {
				healthResult, err := RunHealth(ctx, clientCtx, subscriptionAddresses, providerAddresses, consumerEndpoints, referenceEndpoints, prometheusListenAddr)
				if err != nil {
					utils.LavaFormatError("invalid health run", err)
					healthMetrics.SetFailedRun(runLabel)
				} else {
					CheckHealthResults(healthResult)
					healthMetrics.SetSuccess(runLabel)
				}
			}

			RunHealthCheck(ctx, clientCtx, subscriptionAddresses, providerAddresses, consumerEndpoints, referenceEndpoints, prometheusListenAddr)

			var ticker *time.Ticker
			if interval > 0*time.Second {
				ticker = time.NewTicker(interval) // initially every block we check for a polling time
			} else {
				// never tick again
				ticker = time.NewTicker(1 * time.Second)
				ticker.Stop()
			}
			for {
				select {
				case <-ticker.C:
					RunHealthCheck(ctx, clientCtx, subscriptionAddresses, providerAddresses, consumerEndpoints, referenceEndpoints, prometheusListenAddr)
				case <-ctx.Done():
					utils.LavaFormatInfo("health ctx.Done")
					return nil
				case <-signalChan:
					utils.LavaFormatInfo("health signalChan")
					return nil
				}
			}

		},
	}
	cmdTestHealth.Flags().String(RunLabelFlagName, "", "a label to add to this health checker to differentiate different sources")
	cmdTestHealth.Flags().String(metrics.MetricsListenFlagName, metrics.DisabledFlagOption, "the address to expose prometheus metrics (such as localhost:7779)")
	cmdTestHealth.Flags().Duration(intervalFlagName, intervalDefaultDuration, "the interval duration for the health check, (defaults to 0s) if 0 runs once")
	cmdTestHealth.Flags().Duration(allowedBlockTimeLagFlagName, allowedBlockTimeDefaultLag, "the amount of time one rpc can be behind the most advanced one")
	cmdTestHealth.Flags().Uint64Var(&QueryRetries, QueryRetriesFlagName, QueryRetries, "set the amount of max queries to send every health run to consumers and references")
	viper.BindPFlag(QueryRetriesFlagName, cmdTestHealth.Flags().Lookup(QueryRetriesFlagName)) // bind the flag
	flags.AddQueryFlagsToCmd(cmdTestHealth)
	common.AddRollingLogConfig(cmdTestHealth)
	// add prefix config
	// add monitoring endpoint
	// add the ability to quiet it down
	// add prometheus
	// add health run times
	// add latest health results to prom
	return cmdTestHealth
}
