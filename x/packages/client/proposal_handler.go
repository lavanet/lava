package client

import (
	govclient "github.com/cosmos/cosmos-sdk/x/gov/client"
	"github.com/lavanet/lava/x/packages/client/cli"
	"github.com/lavanet/lava/x/spec/client/rest"
)

// PackagesAddProposalHandler is the param change proposal handler.
var PackagesAddProposalHandler = govclient.NewProposalHandler(cli.NewSubmitPackagesAddProposalTxCmd, rest.ProposalRESTHandler)
