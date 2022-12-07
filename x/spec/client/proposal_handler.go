package client

import (
	govclient "github.com/cosmos/cosmos-sdk/x/gov/client"
	"github.com/lavanet/lava/x/spec/client/cli"
	"github.com/lavanet/lava/x/spec/client/rest"
)

// ProposalHandler is the param change proposal handler.
var SpecAddProposalHandler = govclient.NewProposalHandler(cli.NewSubmitSpecAddProposalTxCmd, rest.ProposalRESTHandler)
