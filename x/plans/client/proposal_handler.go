package client

import (
	govclient "github.com/cosmos/cosmos-sdk/x/gov/client"
	"github.com/lavanet/lava/x/plans/client/cli"
	"github.com/lavanet/lava/x/plans/client/rest"
)

var PlansAddProposalHandler = govclient.NewProposalHandler(cli.NewSubmitPlansAddProposalTxCmd, rest.PlansAddProposalRESTHandler)

var PlansDelProposalHandler = govclient.NewProposalHandler(cli.NewSubmitPlansDelProposalTxCmd, rest.PlansDelProposalRESTHandler)
