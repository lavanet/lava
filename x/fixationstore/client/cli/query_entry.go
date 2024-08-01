package cli

import (
	"fmt"
	"strconv"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/lavanet/lava/v2/x/fixationstore/types"
	"github.com/spf13/cobra"
)

var _ = strconv.Itoa(0)

const (
	HideDataFlagName   = "hide-data"
	StringDataFlagName = "string-data"
)

func CmdEntry() *cobra.Command {
	cmd := &cobra.Command{
		Use: "entry [store-key] [prefix] [key] [block]",
		Short: `Query for a specific entry version. Using the optional --hide-data flag, you can get the entry without 
		its raw data. Using the optional --string-data you can convert the data's raw bytes to a string `,
		Example: `lavad q fixationstore entry [store_key] [prefix] [key] [block] [key]
		lavad q fixationstore entry [store_key] [prefix] [key] [block] [key] --hide-data
		lavad q fixationstore entry [store_key] [prefix] [key] [block] [key] --string-data`,
		Args: cobra.ExactArgs(4),
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			storeKey := args[0]
			prefix := args[1]
			key := args[2]
			blockStr := args[3]
			block, err := strconv.ParseUint(blockStr, 10, 64)
			if err != nil {
				return err
			}

			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			queryClient := types.NewQueryClient(clientCtx)

			// check if the command includes --hide-data
			hideDataFlag := cmd.Flags().Lookup(HideDataFlagName)
			if hideDataFlag == nil {
				return fmt.Errorf("%s flag wasn't found", HideDataFlagName)
			}
			hideData := hideDataFlag.Changed

			// check if the command includes --string-data
			stringDataFlag := cmd.Flags().Lookup(StringDataFlagName)
			if stringDataFlag == nil {
				return fmt.Errorf("%s flag wasn't found", StringDataFlagName)
			}
			stringData := stringDataFlag.Changed

			params := &types.QueryEntryRequest{
				StoreKey:   storeKey,
				Prefix:     prefix,
				Key:        key,
				Block:      block,
				HideData:   hideData,
				StringData: stringData,
			}

			res, err := queryClient.Entry(cmd.Context(), params)
			if err != nil {
				return err
			}

			return clientCtx.PrintProto(res)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)
	cmd.Flags().Bool(HideDataFlagName, false, "hide entry data")
	cmd.Flags().Bool(StringDataFlagName, false, "convert entry data to string")

	return cmd
}
