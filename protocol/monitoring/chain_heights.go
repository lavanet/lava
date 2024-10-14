package monitoring

import (
	"context"
	"fmt"
	"runtime"
	"sync"
	"time"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/spf13/cobra"
	"golang.org/x/sync/semaphore"

	"github.com/lavanet/lava/v3/protocol/lavasession"
	"github.com/lavanet/lava/v3/utils"
	"github.com/lavanet/lava/v3/utils/rand"
	"github.com/lavanet/lava/v3/x/epochstorage/types"
	pairingtypes "github.com/lavanet/lava/v3/x/pairing/types"
)

const (
	chainIDFlag = "chain-id"
	contFlag    = "cont"
)

func CreateChainHeightsCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "chain-heights <chain-id>",
		Short: "Get chain heights from all providers for a specific chain",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			// init
			rand.InitRandomSeed()
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}
			logLevel, err := cmd.Flags().GetString(flags.FlagLogLevel)
			if err != nil {
				return fmt.Errorf("failed to read log level flag: %w", err)
			}
			utils.SetGlobalLoggingLevel(logLevel)

			//
			// Get args
			chainID := args[0]
			cont, err := cmd.Flags().GetUint64(contFlag)
			if err != nil {
				return err
			}

			//
			// Run
			for {
				err := runChainHeights(cmd.Context(), clientCtx, chainID)
				if err != nil {
					return err
				}

				if cont == 0 {
					break
				}

				time.Sleep(time.Duration(cont) * time.Second)
			}

			return nil
		},
	}

	cmd.Flags().Uint64(contFlag, 0, "Continuous mode: seconds to wait before repeating (0 for single run)")
	flags.AddQueryFlagsToCmd(cmd)
	cmd.Flags().String(flags.FlagLogLevel, "info", "The logging level (trace|debug|info|warn|error|fatal|panic)")
	return cmd
}

func runChainHeights(ctx context.Context, clientCtx client.Context, chainID string) error {
	pairingQuerier := pairingtypes.NewQueryClient(clientCtx)

	// Get providers for the chain
	providersResp, err := pairingQuerier.Providers(ctx, &pairingtypes.QueryProvidersRequest{
		ChainID: chainID,
	})
	if err != nil {
		return fmt.Errorf("failed to get providers for chain %s: %w", chainID, err)
	}
	if len(providersResp.StakeEntry) == 0 {
		return fmt.Errorf("no providers found for chain %s", chainID)
	}

	fmt.Printf("Chain Heights for %s:\n", chainID)

	// Determine the number of goroutines to use
	maxGoroutines := runtime.NumCPU() - 2
	if maxGoroutines < 1 {
		maxGoroutines = 1
	}

	// Create a semaphore to limit the number of concurrent goroutines
	sem := semaphore.NewWeighted(int64(maxGoroutines))

	var wg sync.WaitGroup
	results := make(chan string, len(providersResp.StakeEntry))

	for _, provider := range providersResp.StakeEntry {
		if len(provider.Endpoints) == 0 || len(provider.Endpoints[0].GetSupportedServices()) == 0 {
			continue
		}

		wg.Add(1)
		go func(provider types.StakeEntry) {
			defer wg.Done()

			// Acquire semaphore
			if err := sem.Acquire(ctx, 1); err != nil {
				utils.LavaFormatError("Failed to acquire semaphore", err)
				return
			}
			defer sem.Release(1)

			endpoint := provider.Endpoints[0]
			service := endpoint.GetSupportedServices()[0]
			height, err := probeProvider(ctx, endpoint.IPPORT, chainID, service.ApiInterface)
			if err != nil {
				utils.LavaFormatDebug("Error probing provider", utils.LogAttr("provider", provider.Address), utils.LogAttr("error", err))
			} else {
				results <- fmt.Sprintf("  %s: %d", provider.Address, height)
			}
		}(provider)
	}

	// Close the results channel when all goroutines are done
	go func() {
		wg.Wait()
		close(results)
	}()

	// Print results as they come in
	for result := range results {
		if result == "" {
			continue
		}
		fmt.Println(result)
	}

	fmt.Println()

	return nil
}

func probeProvider(ctx context.Context, ipport, chainID, apiInterface string) (int64, error) {
	cswp := lavasession.ConsumerSessionsWithProvider{}
	relayerClient, conn, err := cswp.ConnectRawClientWithTimeout(ctx, ipport)
	if err != nil {
		return 0, err
	}
	defer conn.Close()

	probeReq := &pairingtypes.ProbeRequest{
		Guid:         uint64(rand.Int63()),
		SpecId:       chainID,
		ApiInterface: apiInterface,
	}

	probeResp, err := relayerClient.Probe(ctx, probeReq)
	if err != nil {
		return 0, err
	}

	return probeResp.LatestBlock, nil
}
