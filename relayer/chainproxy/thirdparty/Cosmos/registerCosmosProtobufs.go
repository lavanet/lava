package cosmos_thirdparty

import (
	"context"

	"google.golang.org/grpc"
)

func RegisterLavaProtobufs(s *grpc.Server, cb func(ctx context.Context, method string, reqBody []byte) ([]byte, error)) {

	cosmosbasetendermintv1beta1 := &implementedCosmosBaseTendermintV1beta1{cb: cb}
	pkg.RegisterServiceServer(s, cosmosbasetendermintv1beta1)

	cosmosauthv1beta1 := &implementedCosmosAuthV1beta1{cb: cb}
	pkg.RegisterQueryServer(s, cosmosauthv1beta1)

	cosmosbankv1beta1 := &implementedCosmosBankV1beta1{cb: cb}
	pkg.RegisterQueryServer(s, cosmosbankv1beta1)

	cosmosdistributionv1beta1 := &implementedCosmosDistributionV1beta1{cb: cb}
	pkg.RegisterQueryServer(s, cosmosdistributionv1beta1)

	cosmosevidencev1beta1 := &implementedCosmosEvidenceV1beta1{cb: cb}
	pkg.RegisterQueryServer(s, cosmosevidencev1beta1)

	cosmosfeegrantv1beta1 := &implementedCosmosFeegrantV1beta1{cb: cb}
	pkg.RegisterQueryServer(s, cosmosfeegrantv1beta1)

	cosmosgovv1beta1 := &implementedCosmosGovV1beta1{cb: cb}
	pkg.RegisterQueryServer(s, cosmosgovv1beta1)

	cosmosmintv1beta1 := &implementedCosmosMintV1beta1{cb: cb}
	pkg.RegisterQueryServer(s, cosmosmintv1beta1)

	cosmosparamsv1beta1 := &implementedCosmosParamsV1beta1{cb: cb}
	pkg.RegisterQueryServer(s, cosmosparamsv1beta1)

	cosmosslashingv1beta1 := &implementedCosmosSlashingV1beta1{cb: cb}
	pkg.RegisterQueryServer(s, cosmosslashingv1beta1)

	cosmosstakingv1beta1 := &implementedCosmosStakingV1beta1{cb: cb}
	pkg.RegisterQueryServer(s, cosmosstakingv1beta1)

	cosmostxv1beta1 := &implementedCosmosTxV1beta1{cb: cb}
	pkg.RegisterServiceServer(s, cosmostxv1beta1)

	cosmosupgradev1beta1 := &implementedCosmosUpgradeV1beta1{cb: cb}
	pkg.RegisterQueryServer(s, cosmosupgradev1beta1)

	// this line is used by grpc_scaffolder #Register
}

func RegisterOsmosisProtobufs(s *grpc.Server, cb func(ctx context.Context, method string, reqBody []byte) ([]byte, error)) {

	cosmosauthv1beta1 := &implementedCosmosAuthV1beta1{cb: cb}
	pkg.RegisterQueryServer(s, cosmosauthv1beta1)

	cosmosauthzv1beta1 := &implementedCosmosAuthzV1beta1{cb: cb}
	pkg.RegisterQueryServer(s, cosmosauthzv1beta1)

	cosmosbankv1beta1 := &implementedCosmosBankV1beta1{cb: cb}
	pkg.RegisterQueryServer(s, cosmosbankv1beta1)

	cosmosbasetendermintv1beta1 := &implementedCosmosBaseTendermintV1beta1{cb: cb}
	pkg.RegisterServiceServer(s, cosmosbasetendermintv1beta1)

	cosmosdistributionv1beta1 := &implementedCosmosDistributionV1beta1{cb: cb}
	pkg.RegisterQueryServer(s, cosmosdistributionv1beta1)

	cosmosevidencev1beta1 := &implementedCosmosEvidenceV1beta1{cb: cb}
	pkg.RegisterQueryServer(s, cosmosevidencev1beta1)

	cosmosgovv1beta1 := &implementedCosmosGovV1beta1{cb: cb}
	pkg.RegisterQueryServer(s, cosmosgovv1beta1)

	cosmosparamsv1beta1 := &implementedCosmosParamsV1beta1{cb: cb}
	pkg.RegisterQueryServer(s, cosmosparamsv1beta1)

	cosmosslashingv1beta1 := &implementedCosmosSlashingV1beta1{cb: cb}
	pkg.RegisterQueryServer(s, cosmosslashingv1beta1)

	cosmosstakingv1beta1 := &implementedCosmosStakingV1beta1{cb: cb}
	pkg.RegisterQueryServer(s, cosmosstakingv1beta1)

	cosmostxv1beta1 := &implementedCosmosTxV1beta1{cb: cb}
	pkg.RegisterServiceServer(s, cosmostxv1beta1)

	cosmosupgradev1beta1 := &implementedCosmosUpgradeV1beta1{cb: cb}
	pkg.RegisterQueryServer(s, cosmosupgradev1beta1)

	// this line is used by grpc_scaffolder #Register
}

func RegisterCosmosProtobufs(s *grpc.Server, cb func(ctx context.Context, method string, reqBody []byte) ([]byte, error)) {

	cosmosauthv1beta1 := &implementedCosmosAuthV1beta1{cb: cb}
	pkg.RegisterQueryServer(s, cosmosauthv1beta1)

	cosmosauthzv1beta1 := &implementedCosmosAuthzV1beta1{cb: cb}
	pkg.RegisterQueryServer(s, cosmosauthzv1beta1)

	cosmosbankv1beta1 := &implementedCosmosBankV1beta1{cb: cb}
	pkg.RegisterQueryServer(s, cosmosbankv1beta1)

	cosmosbasetendermintv1beta1 := &implementedCosmosBaseTendermintV1beta1{cb: cb}
	pkg.RegisterServiceServer(s, cosmosbasetendermintv1beta1)

	cosmosdistributionv1beta1 := &implementedCosmosDistributionV1beta1{cb: cb}
	pkg.RegisterQueryServer(s, cosmosdistributionv1beta1)

	cosmosevidencev1beta1 := &implementedCosmosEvidenceV1beta1{cb: cb}
	pkg.RegisterQueryServer(s, cosmosevidencev1beta1)

	cosmosfeegrantv1beta1 := &implementedCosmosFeegrantV1beta1{cb: cb}
	pkg.RegisterQueryServer(s, cosmosfeegrantv1beta1)

	cosmosgovv1beta1 := &implementedCosmosGovV1beta1{cb: cb}
	pkg.RegisterQueryServer(s, cosmosgovv1beta1)

	cosmosmintv1beta1 := &implementedCosmosMintV1beta1{cb: cb}
	pkg.RegisterQueryServer(s, cosmosmintv1beta1)

	cosmosparamsv1beta1 := &implementedCosmosParamsV1beta1{cb: cb}
	pkg.RegisterQueryServer(s, cosmosparamsv1beta1)

	cosmosslashingv1beta1 := &implementedCosmosSlashingV1beta1{cb: cb}
	pkg.RegisterQueryServer(s, cosmosslashingv1beta1)

	cosmosstakingv1beta1 := &implementedCosmosStakingV1beta1{cb: cb}
	pkg.RegisterQueryServer(s, cosmosstakingv1beta1)

	cosmostxv1beta1 := &implementedCosmosTxV1beta1{cb: cb}
	pkg.RegisterServiceServer(s, cosmostxv1beta1)

	cosmosupgradev1beta1 := &implementedCosmosUpgradeV1beta1{cb: cb}
	pkg.RegisterQueryServer(s, cosmosupgradev1beta1)

	// this line is used by grpc_scaffolder #Register
}

func RegisterJunoProtobufs(s *grpc.Server, cb func(ctx context.Context, method string, reqBody []byte) ([]byte, error)) {

	cosmosauthv1beta1 := &implementedCosmosAuthV1beta1{cb: cb}
	pkg.RegisterQueryServer(s, cosmosauthv1beta1)

	cosmosauthzv1beta1 := &implementedCosmosAuthzV1beta1{cb: cb}
	pkg.RegisterQueryServer(s, cosmosauthzv1beta1)

	cosmosbankv1beta1 := &implementedCosmosBankV1beta1{cb: cb}
	pkg.RegisterQueryServer(s, cosmosbankv1beta1)

	cosmosbasetendermintv1beta1 := &implementedCosmosBaseTendermintV1beta1{cb: cb}
	pkg.RegisterServiceServer(s, cosmosbasetendermintv1beta1)

	cosmosdistributionv1beta1 := &implementedCosmosDistributionV1beta1{cb: cb}
	pkg.RegisterQueryServer(s, cosmosdistributionv1beta1)

	cosmosevidencev1beta1 := &implementedCosmosEvidenceV1beta1{cb: cb}
	pkg.RegisterQueryServer(s, cosmosevidencev1beta1)

	cosmosfeegrantv1beta1 := &implementedCosmosFeegrantV1beta1{cb: cb}
	pkg.RegisterQueryServer(s, cosmosfeegrantv1beta1)

	cosmosgovv1beta1 := &implementedCosmosGovV1beta1{cb: cb}
	pkg.RegisterQueryServer(s, cosmosgovv1beta1)

	cosmosparamsv1beta1 := &implementedCosmosParamsV1beta1{cb: cb}
	pkg.RegisterQueryServer(s, cosmosparamsv1beta1)

	cosmosslashingv1beta1 := &implementedCosmosSlashingV1beta1{cb: cb}
	pkg.RegisterQueryServer(s, cosmosslashingv1beta1)

	cosmosstakingv1beta1 := &implementedCosmosStakingV1beta1{cb: cb}
	pkg.RegisterQueryServer(s, cosmosstakingv1beta1)

	cosmostxv1beta1 := &implementedCosmosTxV1beta1{cb: cb}
	pkg.RegisterServiceServer(s, cosmostxv1beta1)

	cosmosupgradev1beta1 := &implementedCosmosUpgradeV1beta1{cb: cb}
	pkg.RegisterQueryServer(s, cosmosupgradev1beta1)

	// this line is used by grpc_scaffolder #Register
}

// this line is used by grpc_scaffolder #Registration
