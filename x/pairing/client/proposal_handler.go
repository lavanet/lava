package client

import (
	govclient "github.com/cosmos/cosmos-sdk/x/gov/client"
	"github.com/lavanet/lava/v5/x/pairing/client/cli"
)

var (
	PairingUnstakeProposal = govclient.NewProposalHandler(cli.NewSubmitUnstakeProposalTxCmd)
	PairingJailProposal    = govclient.NewProposalHandler(cli.NewSubmitJailProposalTxCmd)
	PairingUnjailProposal  = govclient.NewProposalHandler(cli.NewSubmitUnjailProposalTxCmd)
)
