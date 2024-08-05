package common

import (
	"github.com/lavanet/lava/v2/utils"
	"github.com/spf13/cobra"
)

func CreateWarningLogCommandWrapper(innerCommand *cobra.Command, warningMessage string) *cobra.Command {
	return &cobra.Command{
		Use:   innerCommand.Use,
		Short: innerCommand.Short,
		Run: func(cmd *cobra.Command, args []string) {
			utils.LavaFormatWarning(warningMessage, nil)
			_ = innerCommand.Execute()
		},
	}
}
