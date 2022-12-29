package chaintracker

import (
	"context"
	"errors"
	"net"
	"net/http"
	"os"
	"os/signal"
	"sync/atomic"
	"time"

	"github.com/improbable-eng/grpc-web/go/grpcweb"
	"github.com/lavanet/lava/relayer/chainproxy"
	"github.com/lavanet/lava/utils"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
	grpc "google.golang.org/grpc"
)

type ChainTracker struct {
	chainProxy        chainproxy.ChainProxy //used to communicate with the node
	blocksToSave      uint64                // how many finalized blocks to keep
	latestBlockNum    int64
	blockQueueMu      utils.LavaMutex
	blocksQueue       []string // holds all past hashes up until latest block
	forkCallback      *func()  //a function to be called when a fork is detected
	newLatestCallback *func()  //a function to be called when a new block is detected
}

func (cs *ChainTracker) GetLatestBlockNum() int64 {
	return atomic.LoadInt64(&cs.latestBlockNum)
}

func (cs *ChainTracker) setLatestBlockNum(value int64) {
	atomic.StoreInt64(&cs.latestBlockNum, value)
}

// accepts spec.LATEST_BLOCK and spec.NOT_APPLICABLE as input
func (cs *ChainTracker) GetLatestBlockData(fromBlock int64, toBlock int64) (latestBlock int64, requestedHashes map[int64]string, err error) {
	return
}

func (cs *ChainTracker) fetchAllPreviousBlocks(ctx context.Context, latestBlock int64) (err error) {
	return
}

func (cs *ChainTracker) forkChanged(ctx context.Context, newLatestBlock int64) (forked bool, err error) {
	return
}

func (cs *ChainTracker) gotNewBlock(ctx context.Context, newLatestBlock int64) (gotNewBlock bool, err error) {
	return
}

func (cs *ChainTracker) fetchAllPreviousBlocksIfNecessary(ctx context.Context) (err error) {
	return
}

func (cs *ChainTracker) getLatestBlockHash() (latestBlock int64, latestBlockHash string) {
	return
}

func (cs *ChainTracker) start(ctx context.Context, averageBlockTime time.Duration) error {
	// how often to query latest block.
	//TODO: subscribe instead of repeatedly fetching
	ticker := time.NewTicker(averageBlockTime / 3) //sample it three times as much

	// Polls blocks and keeps a queue of them
	go func() {
		for {
			select {
			case <-ticker.C:
				err := cs.fetchAllPreviousBlocksIfNecessary(ctx)
				if err != nil {
					utils.LavaFormatError("failed to fetch all previous blocks and was necessary", err, nil)
				}
			}
		}
	}()

	return nil
}

func (ct *ChainTracker) serve(ctx context.Context, listenAddr string) error {
	if listenAddr == "" {
		return nil
	}
	ctx, cancel := context.WithCancel(ctx)
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, os.Interrupt)
	defer func() {
		signal.Stop(signalChan)
		cancel()
	}()
	lis, err := net.Listen("tcp", listenAddr)
	if err != nil {
		utils.LavaFormatFatal("ChainTracker failure setting up listener", err, &map[string]string{"listenAddr": listenAddr})
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
			utils.LavaFormatInfo("chainTracker Server ctx.Done", nil)
		case <-signalChan:
			utils.LavaFormatInfo("chainTracker Server signalChan", nil)
		}

		shutdownCtx, shutdownRelease := context.WithTimeout(context.Background(), 10*time.Second)
		defer shutdownRelease()

		if err := httpServer.Shutdown(shutdownCtx); err != nil {
			utils.LavaFormatFatal("chainTracker failed to shutdown", err, &map[string]string{})
		}
	}()

	server := &ChainTrackerService{ChainTracker: ct}

	RegisterChainTrackerServiceServer(s, server)

	utils.LavaFormatInfo("Chain Tracker Listening", &map[string]string{"Address": lis.Addr().String()})
	if err := httpServer.Serve(lis); !errors.Is(err, http.ErrServerClosed) {
		utils.LavaFormatFatal("Chain Tracker failed to serve", err, &map[string]string{"Address": lis.Addr().String()})
	}
	return nil
}

func NewChainTracker(ctx context.Context, cp chainproxy.ChainProxy, config ChainTrackerConfig) (chainTracker *ChainTracker, err error) {
	config.validate()
	chainTracker = &ChainTracker{forkCallback: config.ForkCallback, newLatestCallback: config.NewLatestCallback, blocksToSave: config.blocksToSave, chainProxy: cp}
	chainTracker.start(ctx, config.averageBlockTime)
	chainTracker.serve(ctx, config.ServerAddress)
	return
}
