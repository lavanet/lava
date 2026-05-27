// scripts/test/chaos/tools/cache_poison/main.go
//
// Direct cache poisoner — sends a single SetRelay gRPC to the lavap cache
// server with a high `seen_block` value. Used by Tier A chaos scenario
// `a3_cache_poisoning.sh` to provoke the consumer-2 `SetLatestBlockFromSharedState`
// rejection path.
//
// Why a Go tool instead of grpcurl: the cache's proto chain transitively
// imports cosmos-sdk types that aren't trivial to assemble for grpcurl's
// --proto path. Using the project's already-generated `pairingtypes` Go types
// avoids that whole problem; the tool re-uses the same client constructor
// (`pairingtypes.NewRelayerCacheClient`) the real consumer uses, so the wire
// format is byte-identical to production traffic.
//
// Usage:
//
//	go run ./scripts/test/chaos/tools/cache_poison <addr> <chain_id> <seen_block>
//
// Example: poison the chain-wide key for LAV1 with value 1,000,000:
//
//	go run ./scripts/test/chaos/tools/cache_poison 127.0.0.1:20100 LAV1 1000000
//
// The cache server's chain-wide key uses `DefaultExpirationForNonFinalized`
// (500ms) — so a single injection only sticks for half a second. Scenarios
// that need sustained poisoning should call this in a loop (see a3_cache_poisoning.sh).
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strconv"
	"time"

	pairingtypes "github.com/lavanet/lava/v5/x/pairing/types"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func main() {
	if len(os.Args) != 4 {
		fmt.Fprintln(os.Stderr, "usage: cache_poison <addr> <chain_id> <seen_block>")
		os.Exit(2)
	}
	addr := os.Args[1]
	chainID := os.Args[2]
	seenBlock, err := strconv.ParseInt(os.Args[3], 10, 64)
	if err != nil {
		log.Fatalf("invalid seen_block %q: %v", os.Args[3], err)
	}

	// grpc.Dial (deprecated in favor of grpc.NewClient as of grpc-go v1.63, but
	// the project pins v1.62.1 so we use the older API).
	dialCtx, dialCancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer dialCancel()
	conn, err := grpc.DialContext(dialCtx, addr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithBlock())
	if err != nil {
		log.Fatalf("dial %s: %v", addr, err)
	}
	defer conn.Close()

	client := pairingtypes.NewRelayerCacheClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	// The cache server takes max(Response.LatestBlock, SeenBlock) when writing
	// the chain-wide latestBlock key (ecosystem/cache/handlers.go:343,371).
	// We set both to seenBlock so the write succeeds regardless of which side
	// the cache code chooses.
	_, err = client.SetRelay(ctx, &pairingtypes.RelayCacheSet{
		ChainId:        chainID,
		SeenBlock:      seenBlock,
		RequestedBlock: seenBlock,
		// A unique-enough request_hash so we don't collide with real cached
		// entries. The hash is opaque to the chain-wide-key write path.
		RequestHash: []byte(fmt.Sprintf("chaos-test-poison-%d", time.Now().UnixNano())),
		Response: &pairingtypes.RelayReply{
			LatestBlock: seenBlock,
			Data:        []byte("{}"),
		},
		Finalized: false,
	})
	if err != nil {
		log.Fatalf("SetRelay: %v", err)
	}
	fmt.Printf("[cache_poison] injected seen_block=%d on chain=%s @ %s\n", seenBlock, chainID, addr)
}
