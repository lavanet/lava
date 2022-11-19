package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/ignite/cli/ignite/chainconfig"
	"github.com/ignite/cli/ignite/pkg/cache"
	"github.com/ignite/cli/ignite/services/chain"
	pairingTypes "github.com/lavanet/lava/x/pairing/types"
	specTypes "github.com/lavanet/lava/x/spec/types"
	"google.golang.org/grpc"
)

type lavaTest struct {
	wg       sync.WaitGroup
	grpcConn *grpc.ClientConn
	ctxs     map[string]context.Context
}

func (lt *lavaTest) startLava(ctx context.Context) {
	defer lt.wg.Done()
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

func (lt *lavaTest) checkLava() {
	specQueryClient := specTypes.NewQueryClient(lt.grpcConn)

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
}

func (lt *lavaTest) stakeLava() {
	cmd := exec.Cmd{
		Path:   "./scripts/init.sh",
		Args:   []string{"./scripts/init.sh"},
		Stdout: os.Stdout,
		Stderr: os.Stdout,
	}
	cmd.Start()
	cmd.Wait()
}

func (lt *lavaTest) checkStakeLava() {
	specQueryClient := specTypes.NewQueryClient(lt.grpcConn)

	// query all specs
	specQueryRes, err := specQueryClient.SpecAll(context.Background(), &specTypes.QueryAllSpecRequest{})
	if err != nil {
		return
	}

	pairingQueryClient := pairingTypes.NewQueryClient(lt.grpcConn)
	// check if all specs added exist
	if len(specQueryRes.Spec) == 0 {
		fmt.Println("Staking Failed SPEC")
		return
	}
	for _, spec := range specQueryRes.Spec {
		// Query providers

		fmt.Println(spec.GetIndex())
		providerQueryRes, err := pairingQueryClient.Providers(context.Background(), &pairingTypes.QueryProvidersRequest{
			ChainID: spec.GetIndex(),
		})
		if err != nil {
			fmt.Println(err)
			return
		}
		if len(providerQueryRes.StakeEntry) == 0 {
			fmt.Println("Staking Failed PROVIDER")
			return
		}
		for _, providerStakeEntry := range providerQueryRes.StakeEntry {
			// check if number of stakes matches number of providers to be launched
			fmt.Println("provider", providerStakeEntry)
		}

		// Query clients
		clientQueryRes, err := pairingQueryClient.Clients(context.Background(), &pairingTypes.QueryClientsRequest{
			ChainID: spec.GetIndex(),
		})
		if err != nil {
			fmt.Println(err)
			return
		}
		if len(clientQueryRes.StakeEntry) == 0 {
			fmt.Println("Staking Failed CLIENT")
			return
		}
		for _, clientStakeEntry := range clientQueryRes.StakeEntry {
			// check if number of stakes matches number of clients to be launched
			fmt.Println("client", clientStakeEntry)
		}
	}
}

func main() {
	grpcConn, err := grpc.Dial("127.0.0.1:9090", grpc.WithInsecure())
	if err != nil {
		// this check does not get triggered even if server is down
		fmt.Println(err)
		return
	}
	lt := &lavaTest{
		grpcConn: grpcConn,
		ctxs:     make(map[string]context.Context),
	}
	lt.ctxs["lavaMain"] = context.Background()
	lt.wg.Add(1)
	fmt.Println("Starting Lava")
	go lt.startLava(lt.ctxs["lavaMain"])
	lt.checkLava()
	fmt.Println("Starting Lava OK")
	fmt.Println("Staking Lava")
	lt.stakeLava()
	lt.checkStakeLava()
	fmt.Println("Staking Lava OK")
	// startETHProxy()
	// checkETHProxy()
	// RunProviders()
	// RunClients()
	// RunTendermintTest()
	lt.wg.Wait()
}
