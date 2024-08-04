package badgeserver

import (
	"context"
	"net"
	"net/http"
	"strings"

	"github.com/prometheus/client_golang/prometheus/promhttp"

	"google.golang.org/grpc/health/grpc_health_v1"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/server/grpc/gogoreflection"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/improbable-eng/grpc-web/go/grpcweb"
	"github.com/lavanet/lava/v2/app"
	"github.com/lavanet/lava/v2/protocol/chainlib"
	"github.com/lavanet/lava/v2/utils"
	"github.com/lavanet/lava/v2/utils/rand"
	"github.com/lavanet/lava/v2/utils/sigs"
	pairingtypes "github.com/lavanet/lava/v2/x/pairing/types"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
	"google.golang.org/grpc"
)

var (
	// The name of our config file, without the file extension because viper supports many different config file languages.
	DefaultConfigFilename = "badgeserver.yml"
	// The environment variable prefix of all environment variables bound to our command line flags.
	// For example, --number is bound to STING_NUMBER.
	envPrefix = "BADGE"
)

func CreateBadgeServerCobraCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     `badgeserver [config-path] --port=[serving-port] --log-level=[log-level] --chain-id=[chain-id]`,
		Short:   `badgeserver sets up a server to listen for badges requests from the lava sdk and respond with a signed badge`,
		Long:    `badgeserver sets up a server to listen for badges requests from the lava sdk and respond with a signed badge`,
		Example: `badgeserver badgeserver.yml --port=8080 --log-level=debug --chain-id=lava --from user1`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 1 {
				viper.AddConfigPath(args[0])
			}

			viper.SetConfigName(DefaultConfigFilename)
			viper.SetConfigType("yml")
			viper.AddConfigPath(".")
			viper.AddConfigPath("./config")

			rand.InitRandomSeed()
			if err := viper.ReadInConfig(); err != nil {
				return err
			}

			viper.SetEnvPrefix(envPrefix)
			viper.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))
			viper.AutomaticEnv()

			bindFlags(cmd, viper.GetViper())

			logFormat := viper.GetString(flags.FlagLogFormat)
			utils.JsonFormat = logFormat == "json"
			logLevel, err := cmd.Flags().GetString(flags.FlagLogLevel)
			if err != nil {
				utils.LavaFormatFatal("failed to read log level flag", err)
			}
			utils.SetGlobalLoggingLevel(logLevel)

			RunBadgeServer(cmd, viper.GetViper())

			return nil
		},
	}

	flags.AddKeyringFlags(cmd.Flags())
	cmd.Flags().Int("epoch-interval", 30, "--epoch-interval=30")
	cmd.Flags().String("port", "8080", "badge server listening port")
	cmd.Flags().String("metrics-port", "8081", "badge server metrics port")
	cmd.Flags().String(flags.FlagFrom, "", "Name or address of private key with which to sign")
	cmd.Flags().String(flags.FlagChainID, app.Name, "The network chain ID")
	cmd.Flags().String(flags.FlagNode, "tcp://localhost:26657", "<host>:<port> to tendermint rpc interface for this chain")

	return cmd
}

func bindFlags(cmd *cobra.Command, v *viper.Viper) {
	cmd.Flags().VisitAll(func(f *pflag.Flag) {
		configName := strings.ToUpper(strings.ReplaceAll(f.Name, "-", "_"))

		// Apply the viper config value to the flag when the flag is not set and viper has a value
		if f.Changed {
			val, _ := cmd.Flags().GetString(f.Name)
			v.Set(configName, val)
		} else {
			val := v.GetString(configName)
			if val == "" {
				v.Set(configName, f.Value)
			}
		}
	})
}

func RunBadgeServer(cmd *cobra.Command, v *viper.Viper) {
	utils.LavaFormatInfo("Starting the badge server...")

	port := v.GetString(PortFieldName)
	listener, err := net.Listen("tcp", ":"+port)
	if err != nil {
		utils.LavaFormatFatal("Error in open listener", err)
	}
	defaultGeolocation := v.GetInt(DefaultGeolocationFieldName)
	countriesFilePath := v.GetString(CountriesFilePathFieldName)
	ipFilePath := v.GetString(IpFilePathFieldName)
	ipService, err := InitIpService(defaultGeolocation, countriesFilePath, ipFilePath)
	if err != nil {
		utils.LavaFormatFatal("Error initializing ip service", err)
	}
	chainId := v.GetString(LavaChainIDFieldName)

	projectsData := make(GelocationToProjectsConfiguration)
	err = v.UnmarshalKey(ProjectDataFieldName, &projectsData)
	if err != nil {
		utils.LavaFormatFatal("Error in unmarshalling projects data", err)
	}

	ctx := context.Background()
	clientCtx, err := client.GetClientTxContext(cmd)
	if err != nil {
		utils.LavaFormatFatal("Error initiating client to lava", err)
	}

	keyName, err := sigs.GetKeyName(clientCtx)
	if err != nil {
		utils.LavaFormatFatal("failed getting key name from clientCtx", err)
	}

	clientKey, _ := clientCtx.Keyring.Key(keyName)
	pubKey, err := clientKey.GetPubKey()
	if err != nil {
		utils.LavaFormatFatal("failed getting public key from key name", err)
	}

	var pubKeyAddr sdk.AccAddress
	err = pubKeyAddr.Unmarshal(pubKey.Address())
	if err != nil {
		utils.LavaFormatFatal("failed unmarshalling public address", err)
	}

	privKey, err := sigs.GetPrivKey(clientCtx, keyName)
	if err != nil {
		utils.LavaFormatFatal("failed getting private key from key name", err, utils.Attribute{Key: "keyName", Value: keyName})
	}

	lavaChainFetcher := chainlib.NewLavaChainFetcher(ctx, clientCtx)
	stateTracker, err := NewBadgeStateTracker(ctx, clientCtx, lavaChainFetcher, chainId)
	if err != nil {
		utils.LavaFormatFatal("Error initiating state tracker", err)
	}
	// setting stateTracker in server so we can register for spec updates.
	server, err := NewServer(ipService, chainId, projectsData, lavaChainFetcher, clientCtx, pubKeyAddr.String(), privKey)
	if err != nil {
		utils.LavaFormatFatal("Error in server creation", err)
	}

	server.InitializeStateTracker(stateTracker)

	stateTracker.RegisterForEpochUpdates(ctx, server)

	grpcServer := grpc.NewServer()
	grpc_health_v1.RegisterHealthServer(grpcServer, &HealthServer{})
	pairingtypes.RegisterBadgeGeneratorServer(grpcServer, server)
	gogoreflection.Register(grpcServer)

	wrappedServer := grpcweb.WrapServer(grpcServer)
	handler := func(resp http.ResponseWriter, req *http.Request) {
		// Set CORS headers
		resp.Header().Set("Access-Control-Allow-Origin", "*")
		resp.Header().Set("Access-Control-Allow-Headers", "Content-Type,x-grpc-web")

		wrappedServer.ServeHTTP(resp, req)
	}
	httpServer := http.Server{
		Handler: h2c.NewHandler(http.HandlerFunc(handler), &http2.Server{}),
	}
	go func() {
		metricsPort := v.GetString(MetricsPortFieldName)
		http.Handle("/metrics", promhttp.Handler())
		http.ListenAndServe(":"+metricsPort, nil)
	}()

	utils.LavaFormatInfo("Badge server started")
	if err := httpServer.Serve(listener); err != nil {
		utils.LavaFormatFatal("Http Server failed to start", err)
	}
}
