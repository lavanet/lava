package monitoring

import (
	"bytes"
	"context"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/goccy/go-json"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/config"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/lavanet/lava/v2/app"
	"github.com/lavanet/lava/v2/protocol/common"
	"github.com/lavanet/lava/v2/protocol/lavasession"
	"github.com/lavanet/lava/v2/protocol/metrics"
	"github.com/lavanet/lava/v2/utils"
	"github.com/lavanet/lava/v2/utils/rand"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

const (
	allowedBlockTimeDefaultLag                = 30 * time.Second
	intervalDefaultDuration                   = 0 * time.Second
	defaultCUPercentageThreshold              = 0.2
	defaultSubscriptionLeftDays               = 10
	defaultMaxProviderLatency                 = 200 * time.Millisecond
	defaultAlertSuppressionInterval           = 6 * time.Hour
	DefaultSuppressionCountThreshold          = 3
	DisableAlertLogging                       = "disable-alert-logging"
	maxProviderLatencyFlagName                = "max-provider-latency"
	subscriptionLeftTimeFlagName              = "subscription-days-left-alert"
	providerAddressesFlagName                 = "provider_addresses"
	subscriptionAddressesFlagName             = "subscription_addresses"
	intervalFlagName                          = "interval"
	consumerEndpointPropertyName              = "consumer_endpoints"
	referenceEndpointPropertyName             = "reference_endpoints"
	allowedBlockTimeLagFlagName               = "allowed_time_lag"
	queryRetriesFlagName                      = "query-retries"
	alertingWebHookFlagName                   = "alert-webhook-url"
	identifierFlagName                        = "identifier"
	percentageCUFlagName                      = "cu-percent-threshold"
	alertSuppressionIntervalFlagName          = "alert-suppression-interval"
	disableAlertSuppressionFlagName           = "disable-alert-suppression"
	SuppressionCountThresholdFlagName         = "suppression-alert-count-threshold"
	resultsPostAddressFlagName                = "post-results-address"
	resultsPostGUIDFlagName                   = "post-results-guid"
	resultsPostSkipSepcFlagName               = "post-results-skip-spec"
	AllProvidersFlagName                      = "all-providers"
	AllProvidersMarker                        = "all"
	ConsumerGrpcTLSFlagName                   = "consumer-grpc-tls"
	allowInsecureConsumerDialingFlagName      = "allow-insecure-consumer-dialing"
	singleProviderAddressFlagName             = "single-provider-address"
	singleProviderSpecsInterfacesDataFlagName = "single-provider-specs-interfaces-data"
	runOnceAndExitFlagName                    = "run-once-and-exit"
)

func ParseEndpoints(keyName string, viper_endpoints *viper.Viper) (endpoints []*HealthRPCEndpoint, err error) {
	err = viper_endpoints.UnmarshalKey(keyName, &endpoints)
	if err != nil {
		utils.LavaFormatError("could not unmarshal key to endpoints", err, utils.LogAttr("key", keyName), utils.Attribute{Key: "viper_endpoints", Value: viper_endpoints.AllSettings()})
	}
	return
}

func CreateHealthCobraCommand() *cobra.Command {
	cmdTestHealth := &cobra.Command{
		Use:   `health config_file`,
		Short: `start monitoring the health of the protocol processes defined in the config`,
		Long:  `config_file if a path to a yml file`,
		Example: `health config/health_examples/health_config.yml
example health config files can be found in https://github.com/lavanet/lava/v2/blob/main/config/health_examples
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

			utils.LavaFormatDebug("Loaded endpoints", utils.LogAttr("consumer_endpoints", viper.Get(consumerEndpointPropertyName)))

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
			lavasession.AllowInsecureConnectionToProviders = viper.GetBool(lavasession.AllowInsecureConnectionToProvidersFlag)
			if lavasession.AllowInsecureConnectionToProviders {
				utils.LavaFormatWarning("AllowInsecureConnectionToProviders is set to true, this should be used only in development", nil, utils.Attribute{Key: lavasession.AllowInsecureConnectionToProvidersFlag, Value: lavasession.AllowInsecureConnectionToProviders})
			}
			clientCtx = clientCtx.WithChainID(networkChainId)
			rand.InitRandomSeed()
			prometheusListenAddr := viper.GetString(metrics.MetricsListenFlagName)
			providerAddresses := viper.GetStringSlice(providerAddressesFlagName)
			allProviders := viper.GetBool(AllProvidersFlagName)
			singleProvider := viper.GetString(singleProviderAddressFlagName)
			if singleProvider != "" {
				providerAddresses = []string{singleProvider}
				utils.LavaFormatInfo("Health probe provider addresses set to a single provider address", utils.Attribute{Key: "provider", Value: singleProvider})
			} else if allProviders {
				providerAddresses = []string{AllProvidersMarker}
				utils.LavaFormatInfo("Health probe provider addresses set to all")
			}
			singleProviderSpecsInterfacesRawData := viper.GetString(singleProviderSpecsInterfacesDataFlagName)
			var singleProviderSpecsInterfacesData map[string][]string
			if singleProviderSpecsInterfacesRawData != "" {
				if singleProvider == "" {
					utils.LavaFormatFatal("single provider address and single provider specs interfaces data must be set together", nil)
				}
				err := json.Unmarshal([]byte(singleProviderSpecsInterfacesRawData), &singleProviderSpecsInterfacesData)
				if err != nil {
					utils.LavaFormatFatal("Failed to parse singleProviderSpecsInterfacesDataFlagName as JSON", err)
				}
				if len(singleProviderSpecsInterfacesData) == 0 {
					utils.LavaFormatFatal("singleProviderSpecsInterfacesData is empty", nil)
				}
			}
			runOnceAndExit := viper.GetBool(runOnceAndExitFlagName)
			if runOnceAndExit {
				utils.LavaFormatInfo("Run once and exit flag set")
			}
			subscriptionAddresses := viper.GetStringSlice(subscriptionAddressesFlagName)
			keyName := consumerEndpointPropertyName
			consumerEndpoints, _ := ParseEndpoints(keyName, viper.GetViper())
			keyName = referenceEndpointPropertyName
			referenceEndpoints, _ := ParseEndpoints(keyName, viper.GetViper())
			interval := viper.GetDuration(intervalFlagName)
			healthMetrics := metrics.NewHealthMetrics(prometheusListenAddr)
			identifier := viper.GetString(identifierFlagName)
			utils.SetGlobalLoggingLevel(logLevel)
			alertingOptions := AlertingOptions{
				Url:                           viper.GetString(alertingWebHookFlagName),
				Logging:                       !viper.GetBool(DisableAlertLogging),
				Identifier:                    identifier,
				SubscriptionCUPercentageAlert: viper.GetFloat64(percentageCUFlagName),
				SubscriptionLeftTimeAlert:     time.Duration(viper.GetUint64(subscriptionLeftTimeFlagName)) * time.Hour * 24,
				AllowedTimeGapVsReference:     viper.GetDuration(allowedBlockTimeLagFlagName),
				MaxProviderLatency:            viper.GetDuration(maxProviderLatencyFlagName),
				SameAlertInterval:             viper.GetDuration(alertSuppressionIntervalFlagName),
				DisableAlertSuppression:       viper.GetBool(disableAlertSuppressionFlagName),
				SuppressionCounterThreshold:   viper.GetUint64(SuppressionCountThresholdFlagName),
			}
			resultsPostAddress := viper.GetString(resultsPostAddressFlagName)
			resultsPostGUID := viper.GetString(resultsPostGUIDFlagName)
			resultsPostSkipSepc := viper.GetBool(resultsPostSkipSepcFlagName)

			alerting := NewAlerting(alertingOptions)
			RunHealthCheck := func(ctx context.Context,
				clientCtx client.Context,
				subscriptionAddresses []string,
				providerAddresses []string,
				consumerEndpoints []*HealthRPCEndpoint,
				referenceEndpoints []*HealthRPCEndpoint,
				prometheusListenAddr string,
			) {
				utils.LavaFormatInfo("[+] starting health run")
				healthResult, err := RunHealth(ctx, clientCtx, subscriptionAddresses, providerAddresses, consumerEndpoints, referenceEndpoints, prometheusListenAddr, resultsPostGUID, singleProviderSpecsInterfacesData)
				if err != nil {
					utils.LavaFormatError("[-] invalid health run", err)
					if runOnceAndExit {
						os.Exit(0)
					}
					healthMetrics.SetFailedRun(identifier)
				} else {
					if resultsPostAddress != "" {
						if resultsPostSkipSepc {
							healthResult.Specs = nil
						}
						jsonData, err := json.Marshal(healthResult)
						if err != nil {
							utils.LavaFormatError("[-] failed marshaling results", err)
						} else {
							resp, err := http.Post(resultsPostAddress, "application/json", bytes.NewBuffer(jsonData))
							if err != nil {
								utils.LavaFormatError("[-] failed posting health results", err, utils.LogAttr("address", resultsPostAddress))
							} else {
								defer resp.Body.Close()
							}
						}
					}

					utils.LavaFormatInfo("[+] completed health run")
					if runOnceAndExit {
						os.Exit(0)
					}

					healthMetrics.SetLatestBlockData(identifier, healthResult.FormatForLatestBlock())
					alerting.CheckHealthResults(healthResult)
					activeAlerts, unhealthy, healthy := alerting.ActiveAlerts()
					healthMetrics.SetSuccess(identifier)
					healthMetrics.SetAlertResults(identifier, activeAlerts, unhealthy, healthy)
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
				case <-ctx.Done():
					utils.LavaFormatInfo("health ctx.Done")
					return nil
				case <-signalChan:
					utils.LavaFormatInfo("health signalChan")
					return nil
				case <-ticker.C:
					RunHealthCheck(ctx, clientCtx, subscriptionAddresses, providerAddresses, consumerEndpoints, referenceEndpoints, prometheusListenAddr)
				}
			}
		},
	}

	cmdTestHealth.Flags().Bool(lavasession.AllowInsecureConnectionToProvidersFlag, false, "allows connecting to providers without TLS mostly for local health checks inside the same vpc/server")
	cmdTestHealth.Flags().Bool(DisableAlertLogging, false, "set to true to disable printing alerts to stdout")
	cmdTestHealth.Flags().Uint64(SuppressionCountThresholdFlagName, DefaultSuppressionCountThreshold, "how many consecutive alerts need to be triggered so an alert is emitted")
	cmdTestHealth.Flags().Bool(disableAlertSuppressionFlagName, false, "if set to true, this will disable alert suppression and send all alerts every health run")
	cmdTestHealth.Flags().Duration(alertSuppressionIntervalFlagName, defaultAlertSuppressionInterval, "interval of time in which the same alert won't be triggered")
	cmdTestHealth.Flags().Duration(maxProviderLatencyFlagName, defaultMaxProviderLatency, "the maximum allowed provider latency, above which it will alert")
	cmdTestHealth.Flags().Uint64(subscriptionLeftTimeFlagName, defaultSubscriptionLeftDays, "the amount of days left in a subscription to trigger an alert")
	cmdTestHealth.Flags().Float64(percentageCUFlagName, defaultCUPercentageThreshold, "the left cu percentage threshold to trigger a subscription alert")
	cmdTestHealth.Flags().String(identifierFlagName, "", "an identifier to this instance of health added to all alerts, used to differentiate different sources")
	cmdTestHealth.Flags().String(alertingWebHookFlagName, "", "a url to post an alert to")
	cmdTestHealth.Flags().String(metrics.MetricsListenFlagName, metrics.DisabledFlagOption, "the address to expose prometheus metrics (such as localhost:7779)")
	cmdTestHealth.Flags().String(resultsPostAddressFlagName, "", "the address to send the raw results to")
	cmdTestHealth.Flags().String(resultsPostGUIDFlagName, "", "a guid marker to add to the results posted to the results post address")
	cmdTestHealth.Flags().Bool(resultsPostSkipSepcFlagName, false, "enable to send the results without the specs to the results post address")
	cmdTestHealth.Flags().Duration(intervalFlagName, intervalDefaultDuration, "the interval duration for the health check, (defaults to 0s) if 0 runs once")
	cmdTestHealth.Flags().Duration(allowedBlockTimeLagFlagName, allowedBlockTimeDefaultLag, "the amount of time one rpc can be behind the most advanced one")
	cmdTestHealth.Flags().Uint64Var(&QueryRetries, queryRetriesFlagName, QueryRetries, "set the amount of max queries to send every health run to consumers and references")
	cmdTestHealth.Flags().Bool(AllProvidersFlagName, false, "a flag to overwrite the provider addresses with all the currently staked providers")
	cmdTestHealth.Flags().Bool(ConsumerGrpcTLSFlagName, true, "use tls configuration for grpc connections to your consumer")
	cmdTestHealth.Flags().Bool(allowInsecureConsumerDialingFlagName, false, "used to test grpc, to allow insecure (self signed cert).")
	cmdTestHealth.Flags().String(singleProviderAddressFlagName, "", "single provider address in bach32 to override config settings")
	cmdTestHealth.Flags().String(singleProviderSpecsInterfacesDataFlagName, "", "a json of spec:[interfaces...] to make the single provider query faster")

	cmdTestHealth.Flags().Bool(runOnceAndExitFlagName, false, "exit after first run.")

	viper.BindPFlag(queryRetriesFlagName, cmdTestHealth.Flags().Lookup(queryRetriesFlagName)) // bind the flag
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
