package badgegenerator

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
	"github.com/improbable-eng/grpc-web/go/grpcweb"
	"github.com/lavanet/lava/v2/app"
	"github.com/lavanet/lava/v2/protocol/chainlib"
	"github.com/lavanet/lava/v2/utils"
	"github.com/lavanet/lava/v2/utils/rand"
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
	defaultConfigFilename = "badgegenerator"
	// The environment variable prefix of all environment variables bound to our command line flags.
	// For example, --number is bound to STING_NUMBER.
	envPrefix = "BADGE"
)

func CreateBadgeGeneratorCobraCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     `badgegenerator --port=8080 --log-level=debug --lava-rpc=http://127.0.0.1:26657 --lava-grpc=127.0.0.1:9090 --chain-id=lava`,
		Short:   `badgegenerator sets up a server to listen for badges requests from the lava sdk and respond with a signed badge`,
		Long:    `badgegenerator sets up a server to listen for badges requests from the lava sdk and respond with a signed badge`,
		Example: `badgegenerator <flags>`,
		RunE: func(cmd *cobra.Command, args []string) error {
			utils.LavaFormatWarning("This command is deprecated and will be removed in the future. Please use the badgeserver command under lavap instead.", nil)
			v := viper.New()
			v.SetConfigName(defaultConfigFilename)
			v.SetConfigType("yml")
			v.AddConfigPath(".")
			v.AddConfigPath("./config")
			rand.InitRandomSeed()
			if err := v.ReadInConfig(); err != nil {
				// It's okay if there isn't a config file
				if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
					return err
				}
			}

			v.SetEnvPrefix(envPrefix)
			v.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))
			v.AutomaticEnv()

			bindFlags(cmd, v)

			logFormat := viper.GetString(flags.FlagLogFormat)
			utils.JsonFormat = logFormat == "json"
			logLevel, err := cmd.Flags().GetString(flags.FlagLogLevel)
			if err != nil {
				utils.LavaFormatFatal("failed to read log level flag", err)
			}
			utils.SetGlobalLoggingLevel(logLevel)

			RunBadgeServer(cmd, v)

			return nil
		},
	}

	cmd.Flags().String("grpc-url", "", "--grpc-url=127.0.0.1:9090")
	cmd.Flags().Int("epoch-interval", 30, "--epoch-interval=30")
	cmd.Flags().String("port", "8080", "--port=8080")
	cmd.Flags().String("metrics-port", "8081", "--metrics-port=8081")
	cmd.Flags().String(flags.FlagChainID, app.Name, "network chain id")
	cmd.Flags().String(flags.FlagNode, "tcp://localhost:26657", "<host>:<port> to Tendermint RPC interface for this chain")

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
	port := v.GetString(PortEnvironmentVariable)
	listener, err := net.Listen("tcp", ":"+port)
	if err != nil {
		utils.LavaFormatFatal("Error in open listener", err)
	}
	defaultGeolocation := v.GetInt(DefaultGeolocationEnvironmentVariable)
	countriesFilePath := v.GetString(CountriesFilePathEnvironmentVariable)
	ipFilePath := v.GetString(IpFilePathEnvironmentVariable)
	ipService, err := InitIpService(defaultGeolocation, countriesFilePath, ipFilePath)
	if err != nil {
		utils.LavaFormatFatal("Error initializing ip service", err)
	}
	grpcUrl := v.GetString(GrpcUrlEnvironmentVariable)
	chainId := v.GetString(LavaChainIDEnvironmentVariable)
	userData := v.GetString(UserDataEnvironmentVariable)

	server, err := NewServer(ipService, grpcUrl, chainId, userData)
	if err != nil {
		utils.LavaFormatFatal("Error in server creation", err)
	}

	ctx := context.Background()
	clientCtx, err := client.GetClientTxContext(cmd)
	if err != nil {
		utils.LavaFormatFatal("Error initiating client to lava", err)
	}
	lavaChainFetcher := chainlib.NewLavaChainFetcher(ctx, clientCtx)
	stateTracker, err := NewBadgeStateTracker(ctx, clientCtx, lavaChainFetcher, chainId)
	if err != nil {
		utils.LavaFormatFatal("Error initiating state tracker", err)
	}
	// setting stateTracker in server so we can register for spec updates.
	server.InitializeStateTracker(stateTracker)

	stateTracker.RegisterForEpochUpdates(ctx, server)

	s := grpc.NewServer()
	grpc_health_v1.RegisterHealthServer(s, &HealthServer{})
	pairingtypes.RegisterBadgeGeneratorServer(s, server)
	gogoreflection.Register(s)

	wrappedServer := grpcweb.WrapServer(s)
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
		metricsPort := v.GetString(MetricsPortEnvironmentVariable)
		http.Handle("/metrics", promhttp.Handler())
		http.ListenAndServe(":"+metricsPort, nil)
	}()
	if err := httpServer.Serve(listener); err != nil {
		utils.LavaFormatFatal("Http Server failed to start", err)
	}
}
