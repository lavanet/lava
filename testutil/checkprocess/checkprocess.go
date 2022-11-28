package main

import (
	"context"
	"fmt"

	pairingTypes "github.com/lavanet/lava/x/pairing/types"
	specTypes "github.com/lavanet/lava/x/spec/types"
	"google.golang.org/grpc"
)

func main() {
	// Initialize GRPC connection
	grpcConn, err := grpc.Dial(
		"127.0.0.1:9090",    // your gRPC server address.
		grpc.WithInsecure(), // The Cosmos SDK doesn't support any transport security mechanism.
	)
	if err != nil {
		return
	}
	defer grpcConn.Close()
	// Create query client
	specQueryClient := specTypes.NewQueryClient(grpcConn)

	// query all specs
	specQueryRes, err := specQueryClient.SpecAll(context.Background(), &specTypes.QueryAllSpecRequest{})
	if err != nil {
		return
	}

	pairingQueryClient := pairingTypes.NewQueryClient(grpcConn)
	// check if all specs added exist
	for _, spec := range specQueryRes.Spec {
		// Query providers

		fmt.Println(spec.GetIndex())
		providerQueryRes, err := pairingQueryClient.Providers(context.Background(), &pairingTypes.QueryProvidersRequest{
			ChainID: spec.GetIndex(),
		})
		if err != nil {
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
			return
		}
		for _, clientStakeEntry := range clientQueryRes.StakeEntry {
			// check if number of stakes matches number of clients to be launched
			fmt.Println("client", clientStakeEntry)
		}
	}

}
