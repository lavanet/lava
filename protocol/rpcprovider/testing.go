package rpcprovider

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io/ioutil"
	"math/rand"
	"os"
	"os/signal"
	"strings"
	"time"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/tx"
	"github.com/cosmos/cosmos-sdk/version"
	"github.com/gogo/status"
	"github.com/lavanet/lava/app"
	"github.com/lavanet/lava/protocol/common"
	"github.com/lavanet/lava/protocol/lavasession"
	"github.com/lavanet/lava/utils"
	"github.com/lavanet/lava/utils/sigs"
	epochstoragetypes "github.com/lavanet/lava/x/epochstorage/types"
	pairingtypes "github.com/lavanet/lava/x/pairing/types"
	spectypes "github.com/lavanet/lava/x/spec/types"
	"github.com/spf13/cobra"
	grpc "google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	wrapperspb "google.golang.org/protobuf/types/known/wrapperspb"
)

const (
	networkAddressFlag = "network-address"
	certFlag           = "cert-pem"
)

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
	for _, providerEntry := range providerEntries {
		utils.LavaFormatInfo("checking provider entry", utils.Attribute{Key: "chainID", Value: providerEntry.Chain})

		for _, endpoint := range providerEntry.Endpoints {
			checkOneProvider := func() (time.Duration, error) {
				cswp := lavasession.ConsumerSessionsWithProvider{}
				relayerClientPt, conn, err := cswp.ConnectRawClientWithTimeout(ctx, endpoint.IPPORT)
				if err != nil {
					return 0, utils.LavaFormatError("failed connecting to provider endpoint", err, utils.Attribute{Key: "apiInterface", Value: endpoint.UseType}, utils.Attribute{Key: "chainID", Value: providerEntry.Chain}, utils.Attribute{Key: "network address", Value: endpoint.IPPORT})
				}
				defer conn.Close()
				relayerClient := *relayerClientPt
				guid := uint64(rand.Int63())
				relaySentTime := time.Now()
				returned, err := relayerClient.Probe(ctx, &wrapperspb.UInt64Value{Value: guid})
				if err != nil {
					return 0, utils.LavaFormatError("failed probing provider endpoint", err, utils.Attribute{Key: "apiInterface", Value: endpoint.UseType}, utils.Attribute{Key: "chainID", Value: providerEntry.Chain}, utils.Attribute{Key: "network address", Value: endpoint.IPPORT})
				}
				relayLatency := time.Since(relaySentTime)
				if guid != returned.Value {
					return 0, utils.LavaFormatError("probe returned invalid value", err, utils.Attribute{Key: "returnedGuid", Value: returned.Value}, utils.Attribute{Key: "guid", Value: guid}, utils.Attribute{Key: "apiInterface", Value: endpoint.UseType}, utils.Attribute{Key: "chainID", Value: providerEntry.Chain}, utils.Attribute{Key: "network address", Value: endpoint.IPPORT})
				}

				relayRequest := &pairingtypes.RelayRequest{
					RelaySession: &pairingtypes.RelaySession{SpecId: providerEntry.Chain},
					RelayData:    &pairingtypes.RelayPrivateData{ApiInterface: endpoint.UseType},
				}
				_, err = relayerClient.Relay(ctx, relayRequest)
				if err == nil {
					return 0, utils.LavaFormatError("relay Without signature did not error, unexpected", nil, utils.Attribute{Key: "apiInterface", Value: endpoint.UseType}, utils.Attribute{Key: "chainID", Value: providerEntry.Chain}, utils.Attribute{Key: "network address", Value: endpoint.IPPORT})
				}
				code := status.Code(err)
				if code != codes.Code(lavasession.EpochMismatchError.ABCICode()) {
					return 0, utils.LavaFormatError("relay returned unexpected error", err, utils.Attribute{Key: "apiInterface", Value: endpoint.UseType}, utils.Attribute{Key: "chainID", Value: providerEntry.Chain}, utils.Attribute{Key: "network address", Value: endpoint.IPPORT})
				}
				return relayLatency, nil
			}
			probeLatency, err := checkOneProvider()
			if err != nil {
				badChains = append(badChains, providerEntry.Chain+" "+endpoint.UseType)
				continue
			}
			utils.LavaFormatInfo("successfully verified provider endpoint", utils.Attribute{Key: "apiInterface", Value: endpoint.UseType}, utils.Attribute{Key: "chainID", Value: providerEntry.Chain}, utils.Attribute{Key: "network address", Value: endpoint.IPPORT}, utils.Attribute{Key: "probe latency", Value: probeLatency})
			goodChains = append(goodChains, providerEntry.Chain+"-"+endpoint.UseType)
		}
	}
	fmt.Printf("----------------------------------------SUMMARY----------------------------------------\n\nTests Passed:\n%s\n\nTests Failed:\n%s\n\n", strings.Join(goodChains, "; "), strings.Join(badChains, "; "))
	return nil
}

func CreateTestRPCProviderCobraCommand() *cobra.Command {
	cmdTestRPCProvider := &cobra.Command{
		Use:   `rpcprovider {provider_address | --from <wallet>} [--endpoints "listen-ip:listen-port,api-interface,spec-chain-id ..."]`,
		Short: `test an rpc provider by reading stake entries and querying it directly in all api interfaces`,
		Long: `sets up a test-client that probes the rpc provider in all staked chains
need to provider either provider_address or --from wallet_name
optional flag: --endpoints in order to validate provider process before submitting a stake command`,
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
				address = clientKey.GetAddress().String()
			} else {
				address = args[0]
			}
			utils.LavaFormatInfo("RPCProvider Test started", utils.Attribute{Key: "address", Value: address})
			utils.LoggingLevel(logLevel)
			clientCtx = clientCtx.WithChainID(networkChainId)
			txFactory := tx.NewFactoryCLI(clientCtx, cmd.Flags())
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
					if len(splitted) != 3 {
						return fmt.Errorf("invalid argument format in endpoints, must be: HOST:PORT,useType,chainid HOST:PORT,useType,chainid, received: %s", endpointStr)
					}
					endpoint := epochstoragetypes.Endpoint{IPPORT: splitted[0], UseType: splitted[1]}
					providerEntry := epochstoragetypes.StakeEntry{
						Endpoints: []epochstoragetypes.Endpoint{endpoint},
						Chain:     splitted[2],
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
			return startTesting(ctx, clientCtx, txFactory, stakedProviderChains)
		},
	}

	// RPCConsumer command flags
	flags.AddTxFlagsToCmd(cmdTestRPCProvider)
	cmdTestRPCProvider.Flags().String(flags.FlagChainID, app.Name, "network chain id")
	cmdTestRPCProvider.Flags().String(common.EndpointsConfigName, "", "endpoints to check, overwrites reading it from the blockchain")
	return cmdTestRPCProvider
}

func CreateTestRPCProviderCACertificateCobraCommand() *cobra.Command {
	cmdTestProviderCaCert := &cobra.Command{
		Use:     `provider-ca-cert {network-address}`,
		Short:   `test the certificate of an rpc provider`,
		Long:    `test if the rpc provider in the given network address is using the right format of CA certificate`,
		Example: `provider-ca-cert 127.0.0.1:2211`,
		Args:    cobra.RangeArgs(0, 1),
		RunE: func(cmd *cobra.Command, args []string) error {
			// handle flags, pass necessary fields
			networkAddress, err := cmd.Flags().GetString(networkAddressFlag)
			if err != nil {
				return utils.LavaFormatError("cmd.Flags().GetString(networkAddressFlag)", err)
			}
			cert, err := cmd.Flags().GetString(certFlag)
			if err != nil {
				return utils.LavaFormatError("cmd.Flags().GetString(networkAddressFlag)", err)
			}

			ctx := context.Background()
			connectCtx, cancel := context.WithTimeout(ctx, time.Second)
			defer cancel()
			caCert, err := ioutil.ReadFile(cert)
			if err != nil {
				return utils.LavaFormatError("Failed setting up tls certificate from local path", err)
			}

			certPool := x509.NewCertPool()
			if !certPool.AppendCertsFromPEM(caCert) {
				return utils.LavaFormatError("failed to append server certificate", nil)
			}

			creds := credentials.NewTLS(&tls.Config{
				RootCAs: certPool,
			})

			_, err = grpc.DialContext(connectCtx, networkAddress, grpc.WithBlock(), grpc.WithTransportCredentials(creds))
			if err != nil {
				utils.LavaFormatError("Failed to dial network address", err, utils.Attribute{Key: "Address", Value: networkAddress})
				utils.LavaFormatError("It means your provider is not setup correctly or is lacking CA certification", nil)
				return nil
			}
			utils.LavaFormatInfo("Finished dialing network address successfully!")
			utils.LavaFormatInfo("CA certificate is setup correctly!")
			return nil
		},
	}

	cmdTestProviderCaCert.Flags().String(networkAddressFlag, "", "network address")
	err := cmdTestProviderCaCert.MarkFlagRequired(networkAddressFlag)
	if err != nil {
		utils.LavaFormatFatal("MarkFlagRequired Error", err)
	}

	cmdTestProviderCaCert.Flags().String(certFlag, "", "certificate file")
	err = cmdTestProviderCaCert.MarkFlagRequired(certFlag)
	if err != nil {
		utils.LavaFormatFatal("MarkFlagRequired Error", err)
	}

	return cmdTestProviderCaCert
}
