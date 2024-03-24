package cache

import (
	"context"
	"errors"
	"net"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/lavanet/lava/utils/lavaslices"

	"github.com/dgraph-io/ristretto"
	"github.com/improbable-eng/grpc-web/go/grpcweb"
	"github.com/lavanet/lava/utils"
	pairingtypes "github.com/lavanet/lava/x/pairing/types"
	"github.com/spf13/pflag"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
	grpc "google.golang.org/grpc"
)

const (
	ExpirationFlagName               = "expiration"
	ExpirationNonFinalizedFlagName   = "expiration-non-finalized"
	FlagCacheSizeName                = "max-items"
	DefaultExpirationForNonFinalized = 500 * time.Millisecond
	DefaultExpirationTimeFinalized   = time.Hour
	CacheNumCounters                 = 100000000 // expect 10M items
)

type CacheServer struct {
	finalizedCache         *ristretto.Cache
	tempCache              *ristretto.Cache
	ExpirationFinalized    time.Duration
	ExpirationNonFinalized time.Duration
	CacheMetrics           *CacheMetrics
	CacheMaxCost           int64
}

func (cs *CacheServer) InitCache(ctx context.Context, expiration time.Duration, expirationNonFinalized time.Duration, metricsAddr string) {
	cs.ExpirationFinalized = expiration
	cs.ExpirationNonFinalized = expirationNonFinalized
	cache, err := ristretto.NewCache(&ristretto.Config{NumCounters: CacheNumCounters, MaxCost: cs.CacheMaxCost, BufferItems: 64})
	if err != nil {
		utils.LavaFormatFatal("could not create cache", err)
	}
	cs.tempCache = cache

	cache, err = ristretto.NewCache(&ristretto.Config{NumCounters: CacheNumCounters, MaxCost: cs.CacheMaxCost, BufferItems: 64})
	if err != nil {
		utils.LavaFormatFatal("could not create finalized cache", err)
	}
	cs.finalizedCache = cache

	// initialize prometheus
	cs.CacheMetrics = NewCacheMetricsServer(metricsAddr)
}

func (cs *CacheServer) Serve(ctx context.Context,
	listenAddr string,
) {
	ctx, cancel := context.WithCancel(ctx)
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, os.Interrupt)
	defer func() {
		signal.Stop(signalChan)
		cancel()
	}()
	lis, err := net.Listen("tcp", listenAddr)
	if err != nil {
		utils.LavaFormatFatal("cache server failure setting up listener", err, utils.Attribute{Key: "listenAddr", Value: listenAddr})
	}
	s := grpc.NewServer()

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
		select {
		case <-ctx.Done():
			_ = utils.LavaFormatInfo("Cache Server ctx.Done")
		case <-signalChan:
			_ = utils.LavaFormatInfo("Cache Server signalChan")
		}

		shutdownCtx, shutdownRelease := context.WithTimeout(context.Background(), 10*time.Second)
		defer shutdownRelease()

		if err := httpServer.Shutdown(shutdownCtx); err != nil {
			utils.LavaFormatFatal("Cache failed to shutdown", err)
		}
	}()

	Server := &RelayerCacheServer{CacheServer: cs}

	pairingtypes.RegisterRelayerCacheServer(s, Server)

	_ = utils.LavaFormatInfo("Cache Server listening", utils.Attribute{Key: "Address", Value: lis.Addr().String()})
	if err := httpServer.Serve(lis); !errors.Is(err, http.ErrServerClosed) {
		utils.LavaFormatFatal("cache failed to serve", err, utils.Attribute{Key: "Address", Value: lis.Addr().String()})
	}
}

func (cs *CacheServer) ExpirationForChain(averageBlockTimeForChain time.Duration) time.Duration {
	eighthBlock := averageBlockTimeForChain / 8
	return lavaslices.Max([]time.Duration{eighthBlock, cs.ExpirationNonFinalized}) // return the maximum TTL between an eighth block and expiration
}

func Server(
	ctx context.Context,
	listenAddr string,
	metricsAddr string,
	flags *pflag.FlagSet,
) {
	expiration, err := flags.GetDuration(ExpirationFlagName)
	if err != nil {
		utils.LavaFormatFatal("failed to read flag", err, utils.Attribute{Key: "flag", Value: ExpirationFlagName})
	}

	expirationNonFinalized, err := flags.GetDuration(ExpirationNonFinalizedFlagName)
	if err != nil {
		utils.LavaFormatFatal("failed to read flag", err, utils.Attribute{Key: "flag", Value: ExpirationFlagName})
	}

	cacheMaxCost, err := flags.GetInt64(FlagCacheSizeName)
	if err != nil {
		utils.LavaFormatFatal("failed to read flag", err, utils.Attribute{Key: "flag", Value: FlagCacheSizeName})
	}
	cs := CacheServer{CacheMaxCost: cacheMaxCost}

	cs.InitCache(ctx, expiration, expirationNonFinalized, metricsAddr)
	// TODO: have a state tracker
	cs.Serve(ctx, listenAddr)
}
