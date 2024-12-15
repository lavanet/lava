package common

import (
	"time"

	"github.com/lavanet/lava/v4/utils"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

const (
	RollingLogLevelFlag        = "rolling-log-level"
	RollingLogMaxSizeFlag      = "rolling-log-max-size"
	RollingLogMaxAgeFlag       = "rolling-log-max-age"
	RollingLogBackupsFlag      = "rolling-log-backups"
	RollingLogFileLocationFlag = "rolling-log-file-location"
	RollingLogFormat           = "rolling-log-format"
)

const (
	ProcessStartLogText = "Process Started"
	// cors related flags
	CorsHeadersFlag                 = "cors-headers"           // comma separated list of headers, or * for all, default simple cors specification headers
	CorsCredentialsFlag             = "cors-credentials"       // comma separated list of headers, or * for all, default simple cors specification headers
	CorsOriginFlag                  = "cors-origin"            // comma separated list of origins, or * for all, default enabled completely
	CorsMethodsFlag                 = "cors-methods"           // comma separated list of methods, default "GET,POST,PUT,DELETE,OPTIONS"
	CDNCacheDurationFlag            = "cdn-cache-duration"     // how long to cache the preflight response default 24 hours (in seconds) "86400"
	RelaysHealthEnableFlag          = "relays-health-enable"   // enable relays health check, default true
	RelayHealthIntervalFlag         = "relays-health-interval" // interval between each relay health check, default 5m
	SharedStateFlag                 = "shared-state"
	DisableConflictTransactionsFlag = "disable-conflict-transactions" // disable conflict transactions, this will hard the network's data reliability and therefore will harm the service.
	// Disable relay retries when we get node errors.
	// This feature is suppose to help with successful relays in some chains that return node errors on rare race conditions on the serviced chains.
	SetRelayCountOnNodeErrorFlag = "set-retry-count-on-node-error"
	UseStaticSpecFlag            = "use-static-spec" // allows the user to manually load a spec providing a path, this is useful to test spec changes before they hit the blockchain

	// optimizer flags
	SetProviderOptimizerBestTierPickChance    = "set-provider-optimizer-best-tier-pick-chance"
	SetProviderOptimizerWorstTierPickChance   = "set-provider-optimizer-worst-tier-pick-chance"
	SetProviderOptimizerNumberOfTiersToCreate = "set-provider-optimizer-number-of-tiers-to-create"

	// optimizer qos server flags
	OptimizerQosServerAddressFlag          = "optimizer-qos-server-address"    // address of the optimizer qos server to send the qos reports
	OptimizerQosListenFlag                 = "optimizer-qos-listen"            // enable listening for qos reports on metrics endpoint
	OptimizerQosServerPushIntervalFlag     = "optimizer-qos-push-interval"     // interval to push the qos reports to the optimizer qos server
	OptimizerQosServerSamplingIntervalFlag = "optimizer-qos-sampling-interval" // interval to sample the qos reports
	// websocket flags
	RateLimitWebSocketFlag                       = "rate-limit-websocket-requests-per-connection"
	BanDurationForWebsocketRateLimitExceededFlag = "ban-duration-for-websocket-rate-limit-exceeded"
	LimitParallelWebsocketConnectionsPerIpFlag   = "limit-parallel-websocket-connections-per-ip"
	LimitWebsocketIdleTimeFlag                   = "limit-websocket-connection-idle-time"
	RateLimitRequestPerSecondFlag                = "rate-limit-requests-per-second"
	// specification default flags
	AllowMissingApisByDefaultFlagName = "allow-missing-apis-by-default"
)

const (
	defaultRollingLogState        = "off"                 // off
	defaultRollingLogMaxSize      = "100"                 // 100MB
	defaultRollingLogMaxAge       = "1"                   // 1 day
	defaultRollingLogFileBackups  = "3"                   // 3 files of defaultRollingLogMaxSize size
	defaultRollingLogFileLocation = "logs/rollingRPC.log" // logs directory
	defaultRollingLogFormat       = "json"                // defaults to json format
)

// helper struct to propagate flags deeper into the code in an organized manner
type ConsumerCmdFlags struct {
	HeadersFlag                 string        // comma separated list of headers, or * for all, default simple cors specification headers
	CredentialsFlag             string        // access-control-allow-credentials, defaults to "true"
	OriginFlag                  string        // comma separated list of origins, or * for all, default enabled completely
	MethodsFlag                 string        // whether to allow access control headers *, most proxies have their own access control so its not required
	CDNCacheDuration            string        // how long to cache the preflight response defaults 24 hours (in seconds) "86400"
	RelaysHealthEnableFlag      bool          // enables relay health check
	RelaysHealthIntervalFlag    time.Duration // interval for relay health check
	DebugRelays                 bool          // enables debug mode for relays
	DisableConflictTransactions bool          // disable conflict transactions
	StaticSpecPath              string        // path to the spec file, works only when bootstrapping a single chain.
}

// default rolling logs behavior (if enabled) will store 3 files each 100MB for up to 1 day every time.
func AddRollingLogConfig(cmd *cobra.Command) {
	cmd.Flags().String(RollingLogLevelFlag, defaultRollingLogState, "rolling-log info level (off, debug, info, warn, error, fatal)")
	cmd.Flags().String(RollingLogMaxSizeFlag, defaultRollingLogMaxSize, "rolling-log max size in MB")
	cmd.Flags().String(RollingLogMaxAgeFlag, defaultRollingLogMaxAge, "max age in days")
	cmd.Flags().String(RollingLogBackupsFlag, defaultRollingLogFileBackups, "Keep up to X (number) old log files before purging")
	cmd.Flags().String(RollingLogFileLocationFlag, defaultRollingLogFileLocation, "where to store the rolling logs e.g /logs/provider1.log")
	cmd.Flags().String(RollingLogFormat, defaultRollingLogFormat, "rolling log format (json, text)")
}

func SetupRollingLogger() func() {
	return utils.RollingLoggerSetup(
		viper.GetString(RollingLogLevelFlag),
		viper.GetString(RollingLogFileLocationFlag),
		viper.GetString(RollingLogMaxSizeFlag),
		viper.GetString(RollingLogBackupsFlag),
		viper.GetString(RollingLogMaxAgeFlag),
		viper.GetString(RollingLogFormat),
	)
}
