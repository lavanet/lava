package cosmos_thirdparty

import (
	"context"

	auth "cosmossdk.io/api/cosmos/auth/v1beta1"
	authz "cosmossdk.io/api/cosmos/authz/v1beta1"
	bank "cosmossdk.io/api/cosmos/bank/v1beta1"
	tendermint "cosmossdk.io/api/cosmos/base/tendermint/v1beta1"
	dist "cosmossdk.io/api/cosmos/distribution/v1beta1"
	evidence "cosmossdk.io/api/cosmos/evidence/v1beta1"
	feegrant "cosmossdk.io/api/cosmos/feegrant/v1beta1"
	gov "cosmossdk.io/api/cosmos/gov/v1beta1"
	mint "cosmossdk.io/api/cosmos/mint/v1beta1"
	params "cosmossdk.io/api/cosmos/params/v1beta1"
	slashing "cosmossdk.io/api/cosmos/slashing/v1beta1"
	staking "cosmossdk.io/api/cosmos/staking/v1beta1"
	upgrade "cosmossdk.io/api/cosmos/upgrade/v1beta1"
	tx "github.com/lavanet/lava/protocol/chainlib/chainproxy/thirdparty/thirdparty_utils/cosmos/tx/v1beta1"
	"google.golang.org/grpc"
)

func RegisterLavaProtobufs(s *grpc.Server, cb func(ctx context.Context, method string, reqBody []byte) ([]byte, error)) {
	cosmosbasetendermintv1beta1 := &implementedCosmosBaseTendermintV1beta1{cb: cb}
	tendermint.RegisterServiceServer(s, cosmosbasetendermintv1beta1)

	cosmosauthv1beta1 := &implementedCosmosAuthV1beta1{cb: cb}
	auth.RegisterQueryServer(s, cosmosauthv1beta1)

	cosmosbankv1beta1 := &implementedCosmosBankV1beta1{cb: cb}
	bank.RegisterQueryServer(s, cosmosbankv1beta1)

	cosmosdistributionv1beta1 := &implementedCosmosDistributionV1beta1{cb: cb}
	dist.RegisterQueryServer(s, cosmosdistributionv1beta1)

	cosmosevidencev1beta1 := &implementedCosmosEvidenceV1beta1{cb: cb}
	evidence.RegisterQueryServer(s, cosmosevidencev1beta1)

	cosmosfeegrantv1beta1 := &implementedCosmosFeegrantV1beta1{cb: cb}
	feegrant.RegisterQueryServer(s, cosmosfeegrantv1beta1)

	cosmosgovv1beta1 := &implementedCosmosGovV1beta1{cb: cb}
	gov.RegisterQueryServer(s, cosmosgovv1beta1)

	cosmosmintv1beta1 := &implementedCosmosMintV1beta1{cb: cb}
	mint.RegisterQueryServer(s, cosmosmintv1beta1)

	cosmosparamsv1beta1 := &implementedCosmosParamsV1beta1{cb: cb}
	params.RegisterQueryServer(s, cosmosparamsv1beta1)

	cosmosslashingv1beta1 := &implementedCosmosSlashingV1beta1{cb: cb}
	slashing.RegisterQueryServer(s, cosmosslashingv1beta1)

	cosmosstakingv1beta1 := &implementedCosmosStakingV1beta1{cb: cb}
	staking.RegisterQueryServer(s, cosmosstakingv1beta1)

	cosmostxv1beta1 := &implementedCosmosTxV1beta1{cb: cb}
	tx.RegisterServiceServer(s, cosmostxv1beta1)

	cosmosupgradev1beta1 := &implementedCosmosUpgradeV1beta1{cb: cb}
	upgrade.RegisterQueryServer(s, cosmosupgradev1beta1)

	// this line is used by grpc_scaffolder #Register
}

func RegisterOsmosisProtobufs(s *grpc.Server, cb func(ctx context.Context, method string, reqBody []byte) ([]byte, error)) {
	cosmosauthv1beta1 := &implementedCosmosAuthV1beta1{cb: cb}
	auth.RegisterQueryServer(s, cosmosauthv1beta1)

	cosmosauthzv1beta1 := &implementedCosmosAuthzV1beta1{cb: cb}
	authz.RegisterQueryServer(s, cosmosauthzv1beta1)

	cosmosbankv1beta1 := &implementedCosmosBankV1beta1{cb: cb}
	bank.RegisterQueryServer(s, cosmosbankv1beta1)

	cosmosbasetendermintv1beta1 := &implementedCosmosBaseTendermintV1beta1{cb: cb}
	tendermint.RegisterServiceServer(s, cosmosbasetendermintv1beta1)

	cosmosdistributionv1beta1 := &implementedCosmosDistributionV1beta1{cb: cb}
	dist.RegisterQueryServer(s, cosmosdistributionv1beta1)

	cosmosevidencev1beta1 := &implementedCosmosEvidenceV1beta1{cb: cb}
	evidence.RegisterQueryServer(s, cosmosevidencev1beta1)

	cosmosgovv1beta1 := &implementedCosmosGovV1beta1{cb: cb}
	gov.RegisterQueryServer(s, cosmosgovv1beta1)

	cosmosparamsv1beta1 := &implementedCosmosParamsV1beta1{cb: cb}
	params.RegisterQueryServer(s, cosmosparamsv1beta1)

	cosmosslashingv1beta1 := &implementedCosmosSlashingV1beta1{cb: cb}
	slashing.RegisterQueryServer(s, cosmosslashingv1beta1)

	cosmosstakingv1beta1 := &implementedCosmosStakingV1beta1{cb: cb}
	staking.RegisterQueryServer(s, cosmosstakingv1beta1)

	cosmostxv1beta1 := &implementedCosmosTxV1beta1{cb: cb}
	tx.RegisterServiceServer(s, cosmostxv1beta1)

	cosmosupgradev1beta1 := &implementedCosmosUpgradeV1beta1{cb: cb}
	upgrade.RegisterQueryServer(s, cosmosupgradev1beta1)

	// this line is used by grpc_scaffolder #Register
}

func RegisterCosmosProtobufs(s *grpc.Server, cb func(ctx context.Context, method string, reqBody []byte) ([]byte, error)) {
	cosmosauthv1beta1 := &implementedCosmosAuthV1beta1{cb: cb}
	auth.RegisterQueryServer(s, cosmosauthv1beta1)

	cosmosauthzv1beta1 := &implementedCosmosAuthzV1beta1{cb: cb}
	authz.RegisterQueryServer(s, cosmosauthzv1beta1)

	cosmosbankv1beta1 := &implementedCosmosBankV1beta1{cb: cb}
	bank.RegisterQueryServer(s, cosmosbankv1beta1)

	cosmosbasetendermintv1beta1 := &implementedCosmosBaseTendermintV1beta1{cb: cb}
	tendermint.RegisterServiceServer(s, cosmosbasetendermintv1beta1)

	cosmosdistributionv1beta1 := &implementedCosmosDistributionV1beta1{cb: cb}
	dist.RegisterQueryServer(s, cosmosdistributionv1beta1)

	cosmosevidencev1beta1 := &implementedCosmosEvidenceV1beta1{cb: cb}
	evidence.RegisterQueryServer(s, cosmosevidencev1beta1)

	cosmosfeegrantv1beta1 := &implementedCosmosFeegrantV1beta1{cb: cb}
	feegrant.RegisterQueryServer(s, cosmosfeegrantv1beta1)

	cosmosgovv1beta1 := &implementedCosmosGovV1beta1{cb: cb}
	gov.RegisterQueryServer(s, cosmosgovv1beta1)

	cosmosmintv1beta1 := &implementedCosmosMintV1beta1{cb: cb}
	mint.RegisterQueryServer(s, cosmosmintv1beta1)

	cosmosparamsv1beta1 := &implementedCosmosParamsV1beta1{cb: cb}
	params.RegisterQueryServer(s, cosmosparamsv1beta1)

	cosmosslashingv1beta1 := &implementedCosmosSlashingV1beta1{cb: cb}
	slashing.RegisterQueryServer(s, cosmosslashingv1beta1)

	cosmosstakingv1beta1 := &implementedCosmosStakingV1beta1{cb: cb}
	staking.RegisterQueryServer(s, cosmosstakingv1beta1)

	cosmostxv1beta1 := &implementedCosmosTxV1beta1{cb: cb}
	tx.RegisterServiceServer(s, cosmostxv1beta1)

	cosmosupgradev1beta1 := &implementedCosmosUpgradeV1beta1{cb: cb}
	upgrade.RegisterQueryServer(s, cosmosupgradev1beta1)

	// this line is used by grpc_scaffolder #Register
}

func RegisterJunoProtobufs(s *grpc.Server, cb func(ctx context.Context, method string, reqBody []byte) ([]byte, error)) {
	cosmosauthv1beta1 := &implementedCosmosAuthV1beta1{cb: cb}
	auth.RegisterQueryServer(s, cosmosauthv1beta1)

	cosmosauthzv1beta1 := &implementedCosmosAuthzV1beta1{cb: cb}
	authz.RegisterQueryServer(s, cosmosauthzv1beta1)

	cosmosbankv1beta1 := &implementedCosmosBankV1beta1{cb: cb}
	bank.RegisterQueryServer(s, cosmosbankv1beta1)

	cosmosbasetendermintv1beta1 := &implementedCosmosBaseTendermintV1beta1{cb: cb}
	tendermint.RegisterServiceServer(s, cosmosbasetendermintv1beta1)

	cosmosdistributionv1beta1 := &implementedCosmosDistributionV1beta1{cb: cb}
	dist.RegisterQueryServer(s, cosmosdistributionv1beta1)

	cosmosevidencev1beta1 := &implementedCosmosEvidenceV1beta1{cb: cb}
	evidence.RegisterQueryServer(s, cosmosevidencev1beta1)

	cosmosfeegrantv1beta1 := &implementedCosmosFeegrantV1beta1{cb: cb}
	feegrant.RegisterQueryServer(s, cosmosfeegrantv1beta1)

	cosmosgovv1beta1 := &implementedCosmosGovV1beta1{cb: cb}
	gov.RegisterQueryServer(s, cosmosgovv1beta1)

	cosmosparamsv1beta1 := &implementedCosmosParamsV1beta1{cb: cb}
	params.RegisterQueryServer(s, cosmosparamsv1beta1)

	cosmosslashingv1beta1 := &implementedCosmosSlashingV1beta1{cb: cb}
	slashing.RegisterQueryServer(s, cosmosslashingv1beta1)

	cosmosstakingv1beta1 := &implementedCosmosStakingV1beta1{cb: cb}
	staking.RegisterQueryServer(s, cosmosstakingv1beta1)

	cosmostxv1beta1 := &implementedCosmosTxV1beta1{cb: cb}
	tx.RegisterServiceServer(s, cosmostxv1beta1)

	cosmosupgradev1beta1 := &implementedCosmosUpgradeV1beta1{cb: cb}
	upgrade.RegisterQueryServer(s, cosmosupgradev1beta1)

	// this line is used by grpc_scaffolder #Register
}

// this line is used by grpc_scaffolder #Registration
