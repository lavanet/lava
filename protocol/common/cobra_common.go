package common

import (
	"time"

	"github.com/lavanet/lava/v5/utils"
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
	CorsHeadersFlag         = "cors-headers"           // comma separated list of headers, or * for all, default simple cors specification headers
	CorsCredentialsFlag     = "cors-credentials"       // comma separated list of headers, or * for all, default simple cors specification headers
	CorsOriginFlag          = "cors-origin"            // comma separated list of origins, or * for all, default enabled completely
	CorsMethodsFlag         = "cors-methods"           // comma separated list of methods, default "GET,POST,PUT,DELETE,OPTIONS"
	CDNCacheDurationFlag    = "cdn-cache-duration"     // how long to cache the preflight response default 24 hours (in seconds) "86400"
	RelaysHealthEnableFlag  = "relays-health-enable"   // enable relays health check, default true
	RelayHealthIntervalFlag = "relays-health-interval" // interval between each relay health check, default 5m
	SharedStateFlag         = "shared-state"
	// Disable relay retries when we get node errors.
	// This feature is suppose to help with successful relays in some chains that return node errors on rare race conditions on the serviced chains.
	SetRelayCountOnNodeErrorFlag = "set-retry-count-on-node-error"
	// SetRelayRetryLimitFlag controls the maximum number of retry attempts on relay errors
	// (both node errors and protocol errors) for consumers and smart routers.
	SetRelayRetryLimitFlag = "set-relay-retry-limit"
	// BatchNodeErrorOnAny controls batch request error detection for JSON-RPC batch requests
	BatchNodeErrorOnAnyFlag = "batch-node-error-on-any"

	// UseStaticSpecFlag allows loading specs from various sources instead of the blockchain.
	// This flag can be specified multiple times to aggregate specs from multiple sources.
	// Later sources override earlier ones for the same chain ID (last-wins).
	//
	// Supported formats:
	//   - Local file:      --use-static-spec ./specs/eth.json
	//   - Local directory: --use-static-spec ./specs/
	//   - GitHub URL:      --use-static-spec https://github.com/owner/repo/tree/branch/path/to/specs
	//   - GitLab URL:      --use-static-spec https://gitlab.com/owner/repo/-/tree/branch/path/to/specs
	//
	// Multiple local files can be specified either as separate flags or comma-separated:
	//   --use-static-spec file1.json --use-static-spec file2.json
	//   --use-static-spec file1.json,file2.json
	//
	// Example with multiple sources (order matters - later overrides earlier):
	//   --use-static-spec https://github.com/lavanet/lava/tree/main/specs \
	//   --use-static-spec https://gitlab.com/myorg/specs/-/tree/main \
	//   --use-static-spec ./local-overrides/
	//
	// For private GitHub/GitLab repositories, use the corresponding token flag.
	UseStaticSpecFlag = "use-static-spec"

	// GitHubTokenFlag is a GitHub personal access token for accessing private repositories.
	// Also provides higher API rate limits (5,000 requests/hour vs 60 for unauthenticated).
	// Required URL format: https://github.com/{owner}/{repo}/tree/{branch}/{path}
	// Example: --github-token ghp_xxxxxxxxxxxx
	GitHubTokenFlag = "github-token"

	// GitLabTokenFlag is a GitLab personal access token for accessing private repositories.
	// Supports both gitlab.com and self-hosted GitLab instances.
	// The token must have at least "Reporter" role with "read_repository" scope.
	// Required URL format: https://gitlab.com/{owner}/{repo}/-/tree/{branch}/{path}
	// Example: --gitlab-token glpat-xxxxxxxxxxxx
	GitLabTokenFlag = "gitlab-token"

	EpochDurationFlag              = "epoch-duration"         // duration of each epoch for time-based epoch system (standalone mode)
	DefaultEpochDuration           = 30 * time.Minute         // default epoch duration for regular mode (if using time-based epochs)
	StandaloneEpochDuration        = 15 * time.Minute         // default epoch duration for standalone/static provider mode
	EnableSelectionStatsHeaderFlag = "enable-selection-stats" // enable selection stats header for debugging provider selection // allows the user to manually load a spec providing a path, this is useful to test spec changes before they hit the blockchain

	// weighted selection flags (new system replacing tiers)
	UseWeightedSelection                = "use-weighted-provider-selection"         // enable weighted random selection based on composite QoS scores
	ProviderOptimizerAvailabilityWeight = "provider-optimizer-availability-weight"  // weight for availability score (default: 0.4)
	ProviderOptimizerLatencyWeight      = "provider-optimizer-latency-weight"       // weight for latency score (default: 0.3)
	ProviderOptimizerSyncWeight         = "provider-optimizer-sync-weight"          // weight for sync score (default: 0.2)
	ProviderOptimizerStakeWeight        = "provider-optimizer-stake-weight"         // weight for stake (default: 0.1)
	ProviderOptimizerMinSelectionChance = "provider-optimizer-min-selection-chance" // minimum selection probability for any provider (default: 0.01)

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
	SkipWebsocketVerificationFlag                = "skip-websocket-verification"
	// specification default flags
	PeriodicProbeProvidersFlagName         = "enable-periodic-probe-providers"
	PeriodicProbeProvidersIntervalFlagName = "periodic-probe-providers-interval"
	ProbeUpdateWeightFlagName              = "probe-update-weight"

	// batch request size limit
	MaxBatchRequestSizeFlag        = "max-batch-request-size"
	DefaultMaxBatchRequestSize int = 0 // 0 means unlimited

	// DisableBatchRequestRetryFlag prevents batch requests from being retried on consumer/smartrouter side
	DisableBatchRequestRetryFlag = "disable-batch-request-retry"

	ShowProviderEndpointInMetricsFlagName = "show-provider-address-in-metrics"

	MemoryGCThresholdGBFlagName      = "memory-gc-threshold-gb"     // Memory GC threshold in GB (0 = disabled)
	MaxSessionsPerProviderFlagName   = "max-sessions-per-provider"  // Max number of sessions allowed per provider
	DefaultProcessingTimeoutFlagName = "default-processing-timeout" // default timeout for relay processing
	MinRelayTimeoutFlagName          = "min-relay-timeout"          // minimum relay timeout floor (default 1s)

	// ResponseCompressionFlag controls the encoding used by the fiber compress
	// middleware on client-facing responses. Accepted values: "gzip", "brotli", "off".
	// Default is "gzip" because brotli in Go (andybalholm/brotli) costs ~3x more CPU
	// than gzip for similar wire savings on typical JSON-RPC payloads.
	ResponseCompressionFlag    = "response-compression"
	ResponseCompressionGzip    = "gzip"
	ResponseCompressionBrotli  = "brotli"
	ResponseCompressionOff     = "off"
	DefaultResponseCompression = ResponseCompressionGzip
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
	HeadersFlag              string        // comma separated list of headers, or * for all, default simple cors specification headers
	CredentialsFlag          string        // access-control-allow-credentials, defaults to "true"
	OriginFlag               string        // comma separated list of origins, or * for all, default enabled completely
	MethodsFlag              string        // whether to allow access control headers *, most proxies have their own access control so its not required
	CDNCacheDuration         string        // how long to cache the preflight response defaults 24 hours (in seconds) "86400"
	RelaysHealthEnableFlag   bool          // enables relay health check
	RelaysHealthIntervalFlag time.Duration // interval for relay health check
	DebugRelays              bool          // enables debug mode for relays
	StaticSpecPaths          []string      // paths to spec sources (files, directories, or remote URLs). Later entries override earlier for same chain ID.
	GitHubToken              string        // GitHub personal access token for accessing private repositories
	GitLabToken              string        // GitLab personal access token for accessing private repositories
	EpochDuration            time.Duration // duration of each epoch for time-based epoch system (standalone mode)
	EnableSelectionStats     bool          // enables selection stats header for debugging provider selection
	DebugAddress             string        // address for the debug HTTP server, e.g. ":9999". Empty = disabled.
	ResponseCompression      string        // "gzip" (default), "brotli", or "off" — controls client-facing response compression
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
