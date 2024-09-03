package client

import (
	govclient "github.com/cosmos/cosmos-sdk/x/gov/client"
	"github.com/lavanet/lava/v3/x/plans/client/cli"
)

var PlansAddProposalHandler = govclient.NewProposalHandler(cli.NewSubmitPlansAddProposalTxCmd)

var PlansDelProposalHandler = govclient.NewProposalHandler(cli.NewSubmitPlansDelProposalTxCmd)
