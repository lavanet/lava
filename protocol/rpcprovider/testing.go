package rpcprovider

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/config"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/version"
	"github.com/gogo/status"
	"github.com/lavanet/lava/v5/app"
	lvutil "github.com/lavanet/lava/v5/ecosystem/lavavisor/pkg/util"
	"github.com/lavanet/lava/v5/protocol/chainlib/chainproxy"
	"github.com/lavanet/lava/v5/protocol/common"
	"github.com/lavanet/lava/v5/protocol/lavasession"
	"github.com/lavanet/lava/v5/utils"
	"github.com/lavanet/lava/v5/utils/rand"
	"github.com/lavanet/lava/v5/utils/sigs"
	epochstoragetypes "github.com/lavanet/lava/v5/x/epochstorage/types"
	pairingcli "github.com/lavanet/lava/v5/x/pairing/client/cli"
	pairingtypes "github.com/lavanet/lava/v5/x/pairing/types"
	protocoltypes "github.com/lavanet/lava/v5/x/protocol/types"
	spectypes "github.com/lavanet/lava/v5/x/spec/types"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"golang.org/x/exp/slices"
	grpc "google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
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

func PerformCORSCheck(endpoint epochstoragetypes.Endpoint) error {
	utils.LavaFormatDebug("Checking CORS", utils.LogAttr("endpoint", endpoint))
	// Construct the URL for the RPC endpoint
	endpointURL := "https://" + endpoint.IPPORT // Providers must have HTTPS support

	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	client := &http.Client{Transport: tr}

	methods := []string{"OPTIONS", "GET", "POST", "PUT"}
	for _, method := range methods {
		req, err := http.NewRequest(method, endpointURL, nil)
		if err != nil {
			return err
		}

		// Perform the HTTP request
		resp, err := client.Do(req)
		if err != nil {
			return fmt.Errorf("error making %s request to %s: %w", method, endpointURL, err)
		}
		defer resp.Body.Close()

		// Perform CORS Validation
		err = validateCORSHeaders(resp)
		if err != nil {
			return err
		}
	}

	return nil
}

func validateCORSHeaders(resp *http.Response) error {
	// Check for the presence of "Access-Control-Allow-Origin" header
	corsOrigin := resp.Header.Get("Access-Control-Allow-Origin")
	if corsOrigin != "*" {
		return utils.LavaFormatError("CORS check failed. Expected 'Access-Control-Allow-Origin: *' but not found.", nil,
			utils.LogAttr("returned code", resp.StatusCode),
			utils.LogAttr("corsOrigin", corsOrigin),
		)
	}

	// Headers that must be present in "Access-Control-Allow-Headers"
	requiredHeaders := []string{"x-grpc-web"}

	corsHeaders := strings.ToLower(resp.Header.Get("Access-Control-Allow-Headers"))
	for _, requiredHeader := range requiredHeaders {
		if !strings.Contains(corsHeaders, strings.ToLower(requiredHeader)) {
			return utils.LavaFormatError("CORS check failed. Expected 'Access-Control-Allow-Headers' are not present.", nil,
				utils.LogAttr("corsHeaders", corsHeaders),
				utils.LogAttr("requiredHeader", requiredHeader),
			)
		}
	}

	return nil
}

func getEmojiForVerificationStatus(passed bool) string {
	if passed {
		return "✅"
	}
	return "❌"
}

func startTesting(ctx context.Context, clientCtx client.Context, lavaNetworkChainId string, providerEntries []epochstoragetypes.StakeEntry, plainTextConnection bool) error {
	goodChains := []string{}
	badChains := []string{}
	portValidation := []string{}
	verifications := map[string]struct{}{}
	protocolQuerier := protocoltypes.NewQueryClient(clientCtx)
	param, err := protocolQuerier.Params(ctx, &protocoltypes.QueryParamsRequest{})
	if err != nil {
		return err
	}
	lavaVersion := param.GetParams().Version
	targetVersion := lvutil.ParseToSemanticVersion(lavaVersion.ProviderTarget)
	for _, providerEntry := range providerEntries {
		utils.LavaFormatInfo("checking provider entry",
			utils.LogAttr("chainID", providerEntry.Chain),
			utils.LogAttr("endpoints", providerEntry.Endpoints),
		)

		for _, endpoint := range providerEntry.Endpoints {
			checkOneProvider := func(apiInterface string, addon string) (*pairingtypes.ProbeReply, time.Duration, string, int64, error) {
				cswp := lavasession.ConsumerSessionsWithProvider{}
				if portValid := validatePortNumber(endpoint.IPPORT); portValid != "" && !slices.Contains(portValidation, portValid) {
					portValidation = append(portValidation, portValid)
				}
				var conn *grpc.ClientConn
				var err error
				var relayerClient pairingtypes.RelayerClient

				if plainTextConnection {
					utils.LavaFormatWarning("You are using plain text connection (disabled tls), no consumer can connect to it as all consumers use tls. this should be used for testing purposes only", nil)
					conn, err = grpc.DialContext(ctx, endpoint.IPPORT, grpc.WithBlock(), grpc.WithTransportCredentials(insecure.NewCredentials()), grpc.WithDefaultCallOptions(grpc.MaxCallRecvMsgSize(chainproxy.MaxCallRecvMsgSize)))
					if err != nil {
						return nil, 0, "", 0, utils.LavaFormatError("failed connecting to provider endpoint", err,
							utils.LogAttr("apiInterface", apiInterface),
							utils.LogAttr("addon", addon),
							utils.LogAttr("chainID", providerEntry.Chain),
							utils.LogAttr("network address", endpoint.IPPORT),
						)
					}
					relayerClient = pairingtypes.NewRelayerClient(conn)
				} else {
					relayerClient, conn, err = cswp.ConnectRawClientWithTimeout(ctx, endpoint.IPPORT)
					if err != nil {
						if !lavasession.AllowInsecureConnectionToProviders {
							// lets try insecure see if this is the reason
							lavasession.AllowInsecureConnectionToProviders = true
							_, _, err := cswp.ConnectRawClientWithTimeout(ctx, endpoint.IPPORT)
							lavasession.AllowInsecureConnectionToProviders = false
							if err == nil {
								return nil, 0, "", 0, utils.LavaFormatError("provider endpoint is insecure when it should be secure", err,
									utils.LogAttr("apiInterface", apiInterface),
									utils.LogAttr("addon", addon),
									utils.LogAttr("chainID", providerEntry.Chain),
									utils.LogAttr("network address", endpoint.IPPORT),
								)
							}
						}
						return nil, 0, "", 0, utils.LavaFormatError("failed connecting to provider endpoint", err,
							utils.LogAttr("apiInterface", apiInterface),
							utils.LogAttr("addon", addon),
							utils.LogAttr("chainID", providerEntry.Chain),
							utils.LogAttr("network address", endpoint.IPPORT),
						)
					}
				}

				defer conn.Close()
				guid := uint64(rand.Int63())
				relaySentTime := time.Now()
				probeReq := &pairingtypes.ProbeRequest{
					Guid:              guid,
					SpecId:            providerEntry.Chain,
					ApiInterface:      apiInterface,
					WithVerifications: true,
				}
				var trailer metadata.MD
				probeResp, err := relayerClient.Probe(ctx, probeReq, grpc.Trailer(&trailer))
				if err != nil {
					return nil, 0, "", 0, utils.LavaFormatError("failed probing provider endpoint", err,
						utils.LogAttr("apiInterface", apiInterface),
						utils.LogAttr("addon", addon),
						utils.LogAttr("chainID", providerEntry.Chain),
						utils.LogAttr("network address", endpoint.IPPORT),
					)
				}
				utils.LavaFormatDebug("probeResp", utils.LogAttr("probeResp", probeResp.Verifications))

				versions := strings.Join(trailer.Get(common.VersionMetadataKey), ",")
				relayLatency := time.Since(relaySentTime)
				if guid != probeResp.GetGuid() {
					return probeResp, 0, versions, 0, utils.LavaFormatError("probe returned invalid value", err,
						utils.LogAttr("returnedGuid", probeResp.GetGuid()),
						utils.LogAttr("guid", guid),
						utils.LogAttr("apiInterface", apiInterface),
						utils.LogAttr("addon", addon),
						utils.LogAttr("chainID", providerEntry.Chain),
						utils.LogAttr("network address", endpoint.IPPORT),
					)
				}

				// chain id check - lava node chain id should be the same as the one we are probing
				lavaChainIdFromProbeMD := trailer.Get(common.LavaChainIdMetadataKey)
				if len(lavaChainIdFromProbeMD) > 0 {
					lavaChainIdFromProbe := lavaChainIdFromProbeMD[0]
					if lavaChainIdFromProbe != lavaNetworkChainId {
						return probeResp, 0, versions, 0, utils.LavaFormatError("lava chain id from probe does not match the configured network chain id", nil,
							utils.LogAttr("returnedGuid", probeResp.GetGuid()),
							utils.LogAttr("guid", guid),
							utils.LogAttr("apiInterface", apiInterface),
							utils.LogAttr("addon", addon),
							utils.LogAttr("lavaChainIdFromProbe", lavaChainIdFromProbe),
							utils.LogAttr("networkChainId", lavaNetworkChainId),
						)
					}
				}

				// CORS check
				if err := PerformCORSCheck(endpoint); err != nil {
					return probeResp, 0, versions, 0, utils.LavaFormatError("invalid CORS check", err,
						utils.LogAttr("returnedGuid", probeResp.GetGuid()),
						utils.LogAttr("guid", guid),
						utils.LogAttr("apiInterface", apiInterface),
						utils.LogAttr("addon", addon),
						utils.LogAttr("chainID", providerEntry.Chain),
						utils.LogAttr("network address", endpoint.IPPORT),
					)
				}

				relayRequest := &pairingtypes.RelayRequest{
					RelaySession: &pairingtypes.RelaySession{SpecId: providerEntry.Chain},
					RelayData:    &pairingtypes.RelayPrivateData{ApiInterface: apiInterface, Addon: addon},
				}
				_, err = relayerClient.Relay(ctx, relayRequest)
				if err == nil {
					return probeResp, 0, "", 0, utils.LavaFormatError("relay Without signature did not error, unexpected", nil,
						utils.LogAttr("apiInterface", apiInterface),
						utils.LogAttr("addon", addon),
						utils.LogAttr("chainID", providerEntry.Chain),
						utils.LogAttr("network address", endpoint.IPPORT),
					)
				}
				code := status.Code(err)
				if code != codes.Code(lavasession.EpochMismatchError.ABCICode()) {
					return probeResp, 0, versions, 0, utils.LavaFormatError("relay returned unexpected error", err,
						utils.LogAttr("apiInterface", apiInterface),
						utils.LogAttr("addon", addon),
						utils.LogAttr("chainID", providerEntry.Chain),
						utils.LogAttr("network address", endpoint.IPPORT),
					)
				}
				return probeResp, relayLatency, versions, probeResp.GetLatestBlock(), nil
			}
			endpointServices := endpoint.GetSupportedServices()
			if len(endpointServices) == 0 {
				utils.LavaFormatWarning("endpoint has no supported services", nil, utils.LogAttr("endpoint", endpoint))
			}
			for _, endpointService := range endpointServices {
				probeResp, probeLatency, version, latestBlockFromProbe, err := checkOneProvider(endpointService.ApiInterface, endpointService.Addon)
				for _, verification := range probeResp.GetVerifications() {
					verifications["["+getEmojiForVerificationStatus(verification.Passed)+"] "+verification.Name] = struct{}{}
				}
				if err != nil {
					badChains = append(badChains, providerEntry.Chain+" "+endpointService.String())
					continue
				}
				parsedVer := lvutil.ParseToSemanticVersion(strings.TrimPrefix(version, "v"))
				if lvutil.IsVersionLessThan(parsedVer, targetVersion) {
					badChains = append(badChains, providerEntry.Chain+" "+endpointService.String()+" Version:"+version+" should be: "+lavaVersion.ProviderTarget)
					continue
				}
				if lvutil.IsVersionGreaterThan(parsedVer, targetVersion) {
					utils.LavaFormatWarning("provider endpoint version is greater than the target version, this is ok but can lead to unexpected behavior", nil, utils.LogAttr("version", version), utils.LogAttr("targetVersion", lavaVersion.ProviderTarget))
				}
				utils.LavaFormatInfo("successfully verified provider endpoint", utils.LogAttr("version", version),
					utils.LogAttr("enspointService", endpointService),
					utils.LogAttr("chainID", providerEntry.Chain),
					utils.LogAttr("network address", endpoint.IPPORT),
					utils.LogAttr("probe latency", probeLatency),
				)
				goodChains = append(goodChains, providerEntry.Chain+"-"+endpointService.String()+" version: "+version+" latest block: 0x"+strconv.FormatInt(latestBlockFromProbe, 16))
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

	verificationsSlice := []string{}
	for verification := range verifications {
		verificationsSlice = append(verificationsSlice, verification)
	}
	sort.Strings(verificationsSlice)
	fmt.Printf("📄----------------------------------------✨SUMMARY✨----------------------------------------📄\n\n🔵 Tests Passed:\n🔹%s\n\n🔵 Tests Failed:\n🔹%s\n\n🔵 Provider Port Validation:\n🔹%s\n\n🔵 Provider Verifications:\n🔹%s\n", strings.Join(goodChains, "\n🔹"), strings.Join(badChains, "\n🔹"), strings.Join(portValidation, "\n🔹"), strings.Join(verificationsSlice, "\n🔹"))
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

			if networkChainId == app.Name {
				clientTomlConfig, err := config.ReadFromClientConfig(clientCtx)
				if err == nil {
					if clientTomlConfig.ChainID != "" {
						networkChainId = clientTomlConfig.ChainID
					}
				}
			}

			logLevel, err := cmd.Flags().GetString(flags.FlagLogLevel)
			if err != nil {
				utils.LavaFormatFatal("failed to read log level flag", err)
			}
			// setting the insecure option on provider dial, this should be used in development only!
			lavasession.AllowInsecureConnectionToProviders = viper.GetBool(lavasession.AllowInsecureConnectionToProvidersFlag)
			if lavasession.AllowInsecureConnectionToProviders {
				utils.LavaFormatWarning("AllowInsecureConnectionToProviders is set to true, this should be used only in development", nil, utils.LogAttr(lavasession.AllowInsecureConnectionToProvidersFlag, lavasession.AllowInsecureConnectionToProviders))
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
			utils.LavaFormatInfo("RPCProvider Test started", utils.LogAttr("address", address))
			utils.SetGlobalLoggingLevel(logLevel)
			clientCtx = clientCtx.WithChainID(networkChainId)

			utils.LavaFormatInfo("lavad Binary Version: " + version.Version)
			rand.InitRandomSeed()
			resultStatus, err := clientCtx.Client.Status(ctx)
			if err != nil {
				return err
			}

			if resultStatus.NodeInfo.Network != networkChainId {
				return utils.LavaFormatError("network chain id does not match the one in the node", nil,
					utils.LogAttr("networkChainId", networkChainId),
					utils.LogAttr("nodeNetwork", resultStatus.NodeInfo.Network),
				)
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
							return utils.LavaFormatError("failed reading on chain data in order to resolve endpoint", err, utils.LogAttr("endpoint", endpoints[0]))
						}
						endpoints[0].ApiInterfaces = chainInfoResponse.Interfaces
					}
					utils.LavaFormatDebug("endpoints to check", utils.LogAttr("endpoints", endpoints))
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
									utils.LavaFormatWarning("provider is Frozen", nil, utils.LogAttr("chainID", provider.Chain))
								}
								stakedProviderChains = append(stakedProviderChains, provider)
								break
							}
						}
					}
				}
			}
			if len(stakedProviderChains) == 0 {
				utils.LavaFormatError("no active chains for provider", nil, utils.LogAttr("address", address))
			}
			utils.LavaFormatDebug("checking chain entries", utils.LogAttr("stakedProviderChains", stakedProviderChains))
			return startTesting(ctx, clientCtx, networkChainId, stakedProviderChains, viper.GetBool(common.PlainTextConnection))
		},
	}

	// RPCConsumer command flags
	flags.AddTxFlagsToCmd(cmdTestRPCProvider)
	cmdTestRPCProvider.Flags().Bool(lavasession.AllowInsecureConnectionToProvidersFlag, false, "allow insecure provider-dialing. used for development and testing")
	cmdTestRPCProvider.Flags().String(common.EndpointsConfigName, "", "endpoints to check, overwrites reading it from the blockchain")
	cmdTestRPCProvider.Flags().Bool(common.PlainTextConnection, false, "for testing purposes connect provider using plain text (disabled tls)")
	return cmdTestRPCProvider
}
