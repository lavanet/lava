package client

import (
	govclient "github.com/cosmos/cosmos-sdk/x/gov/client"
	"github.com/lavanet/lava/x/plans/client/cli"
	"github.com/lavanet/lava/x/spec/client/rest"
)

var PlansAddProposalHandler = govclient.NewProposalHandler(cli.NewSubmitPlansAddProposalTxCmd, rest.ProposalRESTHandler)
