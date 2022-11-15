package main

import (
	"context"
	"fmt"
	"path/filepath"
	"sync"
	"time"

	"github.com/ignite/cli/ignite/chainconfig"
	"github.com/ignite/cli/ignite/pkg/cache"
	"github.com/ignite/cli/ignite/services/chain"
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
	if err != nil && err != context.Canceled {
		fmt.Println(err)
	}
}

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	wg.Add(1)
	go startLava(ctx)
	time.Sleep(time.Second * 5)
	_ = cancel
	wg.Wait()
}
