module github.com/lavanet/lava

go 1.16

require (
	github.com/cosmos/cosmos-sdk v0.44.5
	github.com/cosmos/ibc-go/v2 v2.0.2
	github.com/google/go-cmp v0.5.7 // indirect
	github.com/spf13/cast v1.4.1
	github.com/stretchr/testify v1.7.0
	github.com/tendermint/starport v0.19.2
	github.com/tendermint/tendermint v0.34.14
	github.com/tendermint/tm-db v0.6.4
	google.golang.org/genproto v0.0.0-20220118154757-00ab72f36ad5 // indirect
	google.golang.org/grpc v1.43.0 // indirect
)

replace (
	github.com/gogo/protobuf => github.com/regen-network/protobuf v1.3.3-alpha.regen.1
	github.com/keybase/go-keychain => github.com/99designs/go-keychain v0.0.0-20191008050251-8e49817e8af4
	google.golang.org/grpc => google.golang.org/grpc v1.33.2
)
