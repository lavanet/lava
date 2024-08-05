package connection

import (
	"context"
	"errors"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/improbable-eng/grpc-web/go/grpcweb"
	"github.com/lavanet/lava/v2/protocol/chainlib"
	"github.com/lavanet/lava/v2/protocol/lavasession"
	"github.com/lavanet/lava/v2/utils"
	pairingtypes "github.com/lavanet/lava/v2/x/pairing/types"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
	"google.golang.org/grpc"
)

const (
	intervalFlagName   = "interval"
	disableTLSFlagName = "disable-tls"
)

func CreateTestConnectionServerCobraCommand() *cobra.Command {
	cmdTestConnectionServer := &cobra.Command{
		Use:     `connection-server [listen_address:port]`,
		Short:   `test an incoming connection`,
		Long:    `sets up a grpc server listning to the probe grpc query and logs connection`,
		Example: `connection-server 127.0.0.1:3333`,
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			utils.LavaFormatInfo("Connection-server started")
			ctx := context.Background()
			listenAddr := args[0]
			ctx, cancel := context.WithCancel(ctx)
			signalChan := make(chan os.Signal, 1)
			signal.Notify(signalChan, os.Interrupt)
			defer func() {
				signal.Stop(signalChan)
				cancel()
			}()

			// GRPC
			lis := chainlib.GetListenerWithRetryGrpc("tcp", listenAddr)
			serverReceiveMaxMessageSize := grpc.MaxRecvMsgSize(1024 * 1024 * 32) // setting receive size to 32mb instead of 4mb default
			grpcServer := grpc.NewServer(serverReceiveMaxMessageSize)

			wrappedServer := grpcweb.WrapServer(grpcServer)
			handler := func(resp http.ResponseWriter, req *http.Request) {
				// Set CORS headers
				resp.Header().Set("Access-Control-Allow-Origin", "*")
				resp.Header().Set("Access-Control-Allow-Headers", "Content-Type, x-grpc-web, lava-sdk-relay-timeout")

				wrappedServer.ServeHTTP(resp, req)
			}

			httpServer := http.Server{
				Handler: h2c.NewHandler(http.HandlerFunc(handler), &http2.Server{}),
			}

			disableTLS := viper.GetBool(disableTLSFlagName)
			var serveExecutor func() error
			if disableTLS {
				utils.LavaFormatWarning("Running with disabled TLS configuration", nil)
				serveExecutor = func() error { return httpServer.Serve(lis) }
			} else {
				NetworkAddressData := lavasession.NetworkAddressData{}
				httpServer.TLSConfig = lavasession.GetTlsConfig(NetworkAddressData)
				serveExecutor = func() error { return httpServer.ServeTLS(lis, "", "") }
			}

			guid := utils.GenerateUniqueIdentifier()
			utils.LavaFormatInfo("running with unique identifier", utils.LogAttr("guid", guid))
			Server := &RelayerConnectionServer{guid: guid}

			pairingtypes.RegisterRelayerServer(grpcServer, Server)

			go func() {
				select {
				case <-ctx.Done():
					_ = utils.LavaFormatInfo("connection-server ctx.Done")
				case <-signalChan:
					_ = utils.LavaFormatInfo("connection-server signalChan")
				}

				shutdownCtx, shutdownRelease := context.WithTimeout(ctx, 10*time.Second)
				defer shutdownRelease()

				if err := httpServer.Shutdown(shutdownCtx); err != nil {
					utils.LavaFormatFatal("connection-server failed to shutdown", err)
				}
			}()

			utils.LavaFormatInfo("connection-server active", utils.Attribute{Key: "address", Value: listenAddr})
			if err := serveExecutor(); !errors.Is(err, http.ErrServerClosed) {
				utils.LavaFormatFatal("connection-server failed to serve", err, utils.Attribute{Key: "Address", Value: lis.Addr().String()})
			}
			utils.LavaFormatInfo("connection-server closed server", utils.Attribute{Key: "address", Value: listenAddr})

			return nil
		},
	}
	cmdTestConnectionServer.Flags().Bool(disableTLSFlagName, false, "used to disable tls in the server, if false will serve secure grpc with a floating certificate")
	return cmdTestConnectionServer
}

func CreateTestConnectionProbeCobraCommand() *cobra.Command {
	cmdTestConnectionProbe := &cobra.Command{
		Use:     `connection-probe [target_address:port]`,
		Short:   `test an a grpc server by probing`,
		Long:    `sets up a grpc client probing the provided grpc server`,
		Example: `connection-probe 127.0.0.1:3333 --interval 2s`,
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			utils.LavaFormatInfo("Connection-prober started")
			ctx, cancel := context.WithCancel(context.Background())
			signalChan := make(chan os.Signal, 1)
			signal.Notify(signalChan, os.Interrupt)
			defer func() {
				signal.Stop(signalChan)
				cancel()
			}()
			lavasession.AllowInsecureConnectionToProviders = viper.GetBool(lavasession.AllowInsecureConnectionToProvidersFlag)
			if lavasession.AllowInsecureConnectionToProviders {
				utils.LavaFormatWarning("AllowInsecureConnectionToProviders is set to true, this should be used only in development", nil, utils.Attribute{Key: lavasession.AllowInsecureConnectionToProvidersFlag, Value: lavasession.AllowInsecureConnectionToProviders})
			}
			address := args[0]
			prober := NewProber(address)
			utils.LavaFormatInfo("[+] making a run")
			err := prober.RunOnce(ctx)
			if err != nil {
				utils.LavaFormatError("failed a run", err)
			}
			interval := viper.GetDuration(intervalFlagName)
			if interval > 0*time.Second {
				ticker := time.NewTicker(interval) // initially every block we check for a polling time
				for {
					select {
					case <-ticker.C:
						utils.LavaFormatInfo("[+] making a run")
						err = prober.RunOnce(ctx)
						utils.LavaFormatError("failed a run", err)
					case <-ctx.Done():
						utils.LavaFormatInfo("prober ctx.Done")
						return nil
					case <-signalChan:
						utils.LavaFormatInfo("prober signalChan")
						return nil
					}
				}
			}
			return err
		},
	}
	cmdTestConnectionProbe.Flags().Bool(lavasession.AllowInsecureConnectionToProvidersFlag, false, "allow insecure provider-dialing. used for development and testing without TLS")
	cmdTestConnectionProbe.Flags().Duration(intervalFlagName, 0, "the interval duration for the health check, (defaults to 0s) if 0 runs once")
	return cmdTestConnectionProbe
}
