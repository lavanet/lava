package main

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/ignite/cli/ignite/chainconfig"
	"github.com/ignite/cli/ignite/pkg/cache"
	"github.com/ignite/cli/ignite/services/chain"
	specTypes "github.com/lavanet/lava/x/spec/types"
	"google.golang.org/grpc"
)

var wg sync.WaitGroup

func startLava(ctx context.Context) {
	defer wg.Done()
	absPath, err := filepath.Abs(".")
	if err != nil {
		fmt.Println(err)
	}

	c, err := chain.New(absPath, chain.LogLevel(chain.LogRegular))
	if err != nil {
		fmt.Println(err)
	}
	cacheRootDir, err := chainconfig.ConfigDirPath()
	if err != nil {
		fmt.Println(err)
	}

	storage, err := cache.NewStorage(filepath.Join(cacheRootDir, "ignite_cache.db"))
	if err != nil {
		fmt.Println(err)
	}

	err = c.Serve(ctx, storage, chain.ServeForceReset())
	if err != nil {
		fmt.Println(err)
	}
}

func main() {
	ctx := context.Background()
	wg.Add(1)
	go startLava(ctx)

	grpcConn, err := grpc.Dial("127.0.0.1:9090", grpc.WithInsecure())
	if err != nil {
		// this check does not get triggered even if server is down
		fmt.Println(err)
		return
	}
	specQueryClient := specTypes.NewQueryClient(grpcConn)

	for {
		// TODO
		// This loop would wait for the lavad server to be up before chain init
		// This one should always end but add a timer
		_, err := specQueryClient.SpecAll(context.Background(), &specTypes.QueryAllSpecRequest{})
		if err != nil && strings.Contains(err.Error(), "rpc error") {
			fmt.Println("Waiting for Lava")
			time.Sleep(time.Second)
		} else if err == nil {
			break
		} else {
			fmt.Println(err)
			return
		}
	}

	fmt.Println("works")
	// ctx.Done()
	wg.Wait()
}
