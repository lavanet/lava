package common

import (
	"github.com/lavanet/lava/utils"
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
	defaultRollingLogState        = "off"                 // off
	defaultRollingLogMaxSize      = "100"                 // 100MB
	defaultRollingLogMaxAge       = "1"                   // 1 day
	defaultRollingLogFileBackups  = "3"                   // 3 files of defaultRollingLogMaxSize size
	defaultRollingLogFileLocation = "logs/rollingRPC.log" // logs directory
	defaultRollingLogFormat       = "json"                // defaults to json format
)

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
