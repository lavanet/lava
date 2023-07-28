package client

import (
	govclient "github.com/cosmos/cosmos-sdk/x/gov/client"
	"github.com/lavanet/lava/x/spec/client/cli"
)

// SpecAddProposalHandler is the param change proposal handler.
var SpecAddProposalHandler = govclient.NewProposalHandler(cli.NewSubmitSpecAddProposalTxCmd)
