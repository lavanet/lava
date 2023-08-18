package lavavisor

import (
	"os"

	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/lavanet/lava/protocol/common"
	"github.com/spf13/cobra"
)

func CreateLavaVisorCreateServiceCobraCommand() *cobra.Command {
	cmdLavavisorCreateService := &cobra.Command{
		Use:   `create-service [service-type: "provider" or "consumer"] [config-file]`,
		Short: "generates service files for each provider/consumer in the config.yml.",
		Long: `The 'create-service' command generates system service files for each provider 
		and consumer specified in the config.yml file. Once these service files are created, 
		the 'lavavisor start' command can utilize them to manage (enable, restart, and check the status of) 
		each service using the 'systemctl' tool. This ensures that each service is properly integrated with 
		the system's service manager, allowing for robust management and monitoring of the LavaVisor services.`,
		Args: cobra.ExactArgs(2),
		Example: `required flags: --geolocation | --from
			optional flags: --log-level 
			lavavisor create-service provider ./config/provider.yml --geolocation 1 --from alice --log-level warn
			lavavisor create-service consumer ./config/consumer.yml --geolocation 1 --from bob --log-level info`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return nil

		},
	}
	flags.AddTxFlagsToCmd(cmdLavavisorCreateService)
	cmdLavavisorCreateService.MarkFlagRequired(flags.FlagFrom)
	cmdLavavisorCreateService.Flags().String(common.GeolocationFlag, os.ExpandEnv("~/"), "Protocol Flags Directory")
	cmdLavavisorCreateService.MarkFlagRequired(common.GeolocationFlag)
	cmdLavavisorCreateService.Flags().String(flags.FlagLogLevel, "debug", "log level")
	return cmdLavavisorCreateService
}
