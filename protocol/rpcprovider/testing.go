package rpcprovider

import (
	"context"
	"fmt"
	"math/rand"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"time"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/tx"
	"github.com/cosmos/cosmos-sdk/version"
	"github.com/gogo/status"
	"github.com/lavanet/lava/protocol/common"
	"github.com/lavanet/lava/protocol/lavasession"
	"github.com/lavanet/lava/utils"
	"github.com/lavanet/lava/utils/sigs"
	epochstoragetypes "github.com/lavanet/lava/x/epochstorage/types"
	pairingcli "github.com/lavanet/lava/x/pairing/client/cli"
	pairingtypes "github.com/lavanet/lava/x/pairing/types"
	spectypes "github.com/lavanet/lava/x/spec/types"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"golang.org/x/exp/slices"
	"google.golang.org/grpc/codes"
)

func validatePortNumber(ipPort string) string {
	utils.LavaFormatDebug("validating iport " + ipPort)
	if lavasession.AllowInsecureConnectionToProviders {
		return "Skipped Port Validation due to Allow Insecure Connection flag"
	}
	// provider-test.lava-cybertron.xyz:443
	if strings.HasSuffix(ipPort, ":443") {
		return ""
	}
	return ipPort
}

func startTesting(ctx context.Context, clientCtx client.Context, txFactory tx.Factory, providerEntries []epochstoragetypes.StakeEntry) error {
	ctx, cancel := context.WithCancel(ctx)
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, os.Interrupt)
	defer func() {
		signal.Stop(signalChan)
		cancel()
	}()
	goodChains := []string{}
	badChains := []string{}
	portValidation := []string{}
	for _, providerEntry := range providerEntries {
		utils.LavaFormatInfo("checking provider entry", utils.Attribute{Key: "chainID", Value: providerEntry.Chain}, utils.Attribute{Key: "endpoints", Value: providerEntry.Endpoints})

		for _, endpoint := range providerEntry.Endpoints {
			checkOneProvider := func(apiInterface string, addon string) (time.Duration, int64, error) {
				cswp := lavasession.ConsumerSessionsWithProvider{}
				if portValid := validatePortNumber(endpoint.IPPORT); portValid != "" && !slices.Contains(portValidation, portValid) {
					portValidation = append(portValidation, portValid)
				}
				relayerClientPt, conn, err := cswp.ConnectRawClientWithTimeout(ctx, endpoint.IPPORT)
				if err != nil {
					return 0, 0, utils.LavaFormatError("failed connecting to provider endpoint", err, utils.Attribute{Key: "apiInterface", Value: apiInterface}, utils.Attribute{Key: "addon", Value: addon}, utils.Attribute{Key: "chainID", Value: providerEntry.Chain}, utils.Attribute{Key: "network address", Value: endpoint.IPPORT})
				}
				defer conn.Close()
				relayerClient := *relayerClientPt
				guid := uint64(rand.Int63())
				relaySentTime := time.Now()
				probeReq := &pairingtypes.ProbeRequest{
					Guid:         guid,
					SpecId:       providerEntry.Chain,
					ApiInterface: apiInterface,
				}
				probeResp, err := relayerClient.Probe(ctx, probeReq)
				if err != nil {
					return 0, 0, utils.LavaFormatError("failed probing provider endpoint", err, utils.Attribute{Key: "apiInterface", Value: apiInterface}, utils.Attribute{Key: "addon", Value: addon}, utils.Attribute{Key: "chainID", Value: providerEntry.Chain}, utils.Attribute{Key: "network address", Value: endpoint.IPPORT})
				}
				relayLatency := time.Since(relaySentTime)
				if guid != probeResp.GetGuid() {
					return 0, 0, utils.LavaFormatError("probe returned invalid value", err, utils.Attribute{Key: "returnedGuid", Value: probeResp.GetGuid()}, utils.Attribute{Key: "guid", Value: guid}, utils.Attribute{Key: "apiInterface", Value: apiInterface}, utils.Attribute{Key: "addon", Value: addon}, utils.Attribute{Key: "chainID", Value: providerEntry.Chain}, utils.Attribute{Key: "network address", Value: endpoint.IPPORT})
				}

				relayRequest := &pairingtypes.RelayRequest{
					RelaySession: &pairingtypes.RelaySession{SpecId: providerEntry.Chain},
					RelayData:    &pairingtypes.RelayPrivateData{ApiInterface: apiInterface, Addon: addon},
				}
				_, err = relayerClient.Relay(ctx, relayRequest)
				if err == nil {
					return 0, 0, utils.LavaFormatError("relay Without signature did not error, unexpected", nil, utils.Attribute{Key: "apiInterface", Value: apiInterface}, utils.Attribute{Key: "addon", Value: addon}, utils.Attribute{Key: "chainID", Value: providerEntry.Chain}, utils.Attribute{Key: "network address", Value: endpoint.IPPORT})
				}
				code := status.Code(err)
				if code != codes.Code(lavasession.EpochMismatchError.ABCICode()) {
					return 0, 0, utils.LavaFormatError("relay returned unexpected error", err, utils.Attribute{Key: "apiInterface", Value: apiInterface}, utils.Attribute{Key: "addon", Value: addon}, utils.Attribute{Key: "chainID", Value: providerEntry.Chain}, utils.Attribute{Key: "network address", Value: endpoint.IPPORT})
				}
				return relayLatency, probeResp.GetLatestBlock(), nil
			}
			endpointServices := endpoint.GetSupportedServices()
			if len(endpointServices) == 0 {
				utils.LavaFormatWarning("endpoint has no supported services", nil, utils.Attribute{Key: "endpoint", Value: endpoint})
			}
			for _, endpointService := range endpointServices {
				probeLatency, latestBlockFromProbe, err := checkOneProvider(endpointService.ApiInterface, endpointService.Addon)
				if err != nil {
					badChains = append(badChains, providerEntry.Chain+" "+endpointService.String())
					continue
				}
				utils.LavaFormatInfo("successfully verified provider endpoint", utils.Attribute{Key: "enspointService", Value: endpointService}, utils.Attribute{Key: "chainID", Value: providerEntry.Chain}, utils.Attribute{Key: "network address", Value: endpoint.IPPORT}, utils.Attribute{Key: "probe latency", Value: probeLatency})
				goodChains = append(goodChains, providerEntry.Chain+"-"+endpointService.String()+" latest block: 0x"+strconv.FormatInt(latestBlockFromProbe, 16))
			}
		}
	}
	if len(badChains) == 0 {
		badChains = []string{"None 🎉! all tests passed ✅"}
	}
	if len(portValidation) == 0 {
		portValidation = []string{"✅ All Ports are valid! ✅"}
	} else {
		portValidation = append([]string{
			"Some provider ports may not be set to 443, which can lead to connection issues with your provider, as some routers might block other ports.",
			"We recommend changing the port configuration of these URLs to 443",
			"Misconfigured URLs:",
		}, portValidation...)
	}
	fmt.Printf("📄----------------------------------------✨SUMMARY✨----------------------------------------📄\n\n🔵 Tests Passed:\n🔹%s\n\n🔵 Tests Failed:\n🔹%s\n\n🔵 Provider Port Validation:\n🔹%s\n\n", strings.Join(goodChains, "\n🔹"), strings.Join(badChains, "\n🔹"), strings.Join(portValidation, "\n🔹"))
	return nil
}

func CreateTestRPCProviderCobraCommand() *cobra.Command {
	cmdTestRPCProvider := &cobra.Command{
		Use:   `rpcprovider {provider_address | --from <wallet>} [--endpoints "listen-ip:listen-port,[api-interface|addon,...],spec-chain-id ..."]`,
		Short: `test an rpc provider by reading stake entries and querying it directly in all api interfaces`,
		Long: `sets up a test-client that probes the rpc provider in all staked chains
need to provider either provider_address or --from wallet_name
optional flag: --endpoints in order to validate provider process before submitting a stake command
endpoints is a space separated list of endpoint,
each endpoint is: listen-ip:listen-port(the url),[optional: the api interfaces and addon to check],spec-id(the spec identifier to test)`,
		Example: `rpcprovider lava@myprovideraddress
rpcprovider --from providerWallet
rpcprovider --from providerWallet --endpoints "provider-public-grpc:port,jsonrpc,ETH1 provider-public-grpc:port,rest,LAV1"`,
		Args: cobra.RangeArgs(0, 1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}
			// handle flags, pass necessary fields
			ctx := context.Background()
			networkChainId, err := cmd.Flags().GetString(flags.FlagChainID)
			if err != nil {
				return err
			}
			logLevel, err := cmd.Flags().GetString(flags.FlagLogLevel)
			if err != nil {
				utils.LavaFormatFatal("failed to read log level flag", err)
			}
			// setting the insecure option on provider dial, this should be used in development only!
			lavasession.AllowInsecureConnectionToProviders = viper.GetBool(lavasession.AllowInsecureConnectionToProvidersFlag)
			if lavasession.AllowInsecureConnectionToProviders {
				utils.LavaFormatWarning("AllowInsecureConnectionToProviders is set to true, this should be used only in development", nil, utils.Attribute{Key: lavasession.AllowInsecureConnectionToProvidersFlag, Value: lavasession.AllowInsecureConnectionToProviders})
			}

			var address string
			if len(args) == 0 {
				keyName, err := sigs.GetKeyName(clientCtx)
				if err != nil {
					utils.LavaFormatFatal("failed getting key name from clientCtx, either provide the address in an argument or verify the --from wallet exists", err)
				}
				clientKey, err := clientCtx.Keyring.Key(keyName)
				if err != nil {
					return err
				}
				tmpAddr, err := clientKey.GetAddress()
				if err != nil {
					return err
				}
				address = tmpAddr.String()
			} else {
				address = args[0]
			}
			utils.LavaFormatInfo("RPCProvider Test started", utils.Attribute{Key: "address", Value: address})
			utils.LoggingLevel(logLevel)
			clientCtx = clientCtx.WithChainID(networkChainId)
			txFactory, err := tx.NewFactoryCLI(clientCtx, cmd.Flags())
			if err != nil {
				utils.LavaFormatFatal("failed to create txFactory", err)
			}

			utils.LavaFormatInfo("lavad Binary Version: " + version.Version)
			rand.Seed(time.Now().UnixNano())
			resultStatus, err := clientCtx.Client.Status(ctx)
			if err != nil {
				return err
			}
			currentBlock := resultStatus.SyncInfo.LatestBlockHeight
			// get all chains provider is serving and their endpoints
			specQuerier := spectypes.NewQueryClient(clientCtx)
			allChains, err := specQuerier.ShowAllChains(ctx, &spectypes.QueryShowAllChainsRequest{})
			if err != nil {
				return utils.LavaFormatError("failed getting key name from clientCtx, either provide the address in an argument or verify the --from wallet exists", err)
			}
			pairingQuerier := pairingtypes.NewQueryClient(clientCtx)
			stakedProviderChains := []epochstoragetypes.StakeEntry{}
			endpointConf, err := cmd.Flags().GetString(common.EndpointsConfigName)
			if err != nil {
				utils.LavaFormatFatal("failed to read endpoints flag", err)
			}
			if endpointConf != "" {
				tmpArg := strings.Fields(endpointConf)
				for _, endpointStr := range tmpArg {
					splitted := strings.Split(endpointStr, ",")
					if len(splitted) < 2 {
						return fmt.Errorf("invalid argument format in endpoints, must be: HOST:PORT,[optional:apiInterface|addon...],chainid HOST:PORT,[optional:apiInterface|addon...],chainid, received: %s", endpointStr)
					}
					chainID := splitted[len(splitted)-1]
					// add dummy geoloc
					splitted[0] = splitted[0] + "," + "1" // add dummy geoLoc

					endpointsToParseJoin := []string{strings.Join(splitted[:len(splitted)-1], ",")}
					endpoints, _, err := pairingcli.HandleEndpointsAndGeolocationArgs(endpointsToParseJoin, "*")
					if err != nil {
						return err
					}
					if len(endpoints) == 0 {
						return fmt.Errorf("returned empty endpoints list from HandleEndpointsAndGeolocationArgs parsing: %s", endpointsToParseJoin)
					}
					if len(endpoints[0].ApiInterfaces) == 0 {
						// need to read required apiInterfaces from on chain
						chainInfoResponse, err := specQuerier.ShowChainInfo(ctx, &spectypes.QueryShowChainInfoRequest{
							ChainName: chainID,
						})
						if err != nil {
							return utils.LavaFormatError("failed reading on chain data in order to resolve endpoint", err, utils.Attribute{Key: "endpoint", Value: endpoints[0]})
						}
						endpoints[0].ApiInterfaces = chainInfoResponse.Interfaces
					}
					utils.LavaFormatDebug("endpoints to check", utils.Attribute{Key: "endpoints", Value: endpoints})
					providerEntry := epochstoragetypes.StakeEntry{
						Endpoints:   endpoints,
						Chain:       chainID,
						Geolocation: 1,
					}
					stakedProviderChains = append(stakedProviderChains, providerEntry)
				}
			} else {
				for _, chainStructInfo := range allChains.ChainInfoList {
					chainID := chainStructInfo.ChainID
					response, err := pairingQuerier.Providers(ctx, &pairingtypes.QueryProvidersRequest{
						ChainID:    chainID,
						ShowFrozen: true,
					})
					if err == nil && len(response.StakeEntry) > 0 {
						for _, provider := range response.StakeEntry {
							if provider.Address == address {
								if provider.StakeAppliedBlock > uint64(currentBlock+1) {
									utils.LavaFormatWarning("provider is Frozen", nil, utils.Attribute{Key: "chainID", Value: provider.Chain})
								}
								stakedProviderChains = append(stakedProviderChains, provider)
								break
							}
						}
					}
				}
			}
			if len(stakedProviderChains) == 0 {
				utils.LavaFormatError("no active chains for provider", nil, utils.Attribute{Key: "address", Value: address})
			}
			utils.LavaFormatDebug("checking chain entries", utils.Attribute{Key: "stakedProviderChains", Value: stakedProviderChains})
			return startTesting(ctx, clientCtx, txFactory, stakedProviderChains)
		},
	}

	// RPCConsumer command flags
	flags.AddTxFlagsToCmd(cmdTestRPCProvider)
	cmdTestRPCProvider.Flags().Bool(lavasession.AllowInsecureConnectionToProvidersFlag, false, "allow insecure provider-dialing. used for development and testing")
	cmdTestRPCProvider.Flags().String(common.EndpointsConfigName, "", "endpoints to check, overwrites reading it from the blockchain")
	return cmdTestRPCProvider
}
