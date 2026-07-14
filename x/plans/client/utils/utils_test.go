package utils

import (
	"testing"

	"github.com/lavanet/lava/v5/x/plans/types"
	"github.com/stretchr/testify/require"
)

// The cookbook gateway-full.json proposal is an update of the gateway.json
// proposal (both propose the same "gateway" plan). This test parses both
// files and documents exactly what the update changes.
func TestDiffCookbookGatewayPlans(t *testing.T) {
	original, err := ParsePlansAddProposalJSON("../../../../cookbook/plans/gateway.json")
	require.NoError(t, err)
	require.Len(t, original.Proposal.Plans, 1)

	updated, err := ParsePlansAddProposalJSON("../../../../cookbook/plans/gateway-full.json")
	require.NoError(t, err)
	require.Len(t, updated.Proposal.Plans, 1)

	// same plan index: the second proposal is an update of the first
	require.Equal(t, original.Proposal.Plans[0].Index, updated.Proposal.Plans[0].Index)

	expected := []string{
		"allowed_buyers: added [lava@1x07x0krj7h00jejrcp35kvlc8tpa44dza76dw8], removed [lava@1w5ck6qa34p2fwfue0fc3wf9n44agcg6r3pm5j5]",
		"plan_policy.chain_policies: added NEAR{requirements: [jsonrpc/POST ext=[archive] mixed]}",
		"plan_policy.chain_policies: added STRK{requirements: [jsonrpc/POST addon=trace mixed]}",
		"plan_policy.chain_policies: added ETH1{requirements: [jsonrpc/POST ext=[archive] mixed]}",
		"plan_policy.chain_policies: added EVMOS{requirements: [jsonrpc/POST ext=[archive] mixed, rest/GET ext=[archive] mixed, grpc ext=[archive] mixed, tendermintrpc ext=[archive] mixed]}",
		"plan_policy.chain_policies: added AXELAR{requirements: [rest/GET ext=[archive] mixed, tendermintrpc ext=[archive] mixed, grpc ext=[archive] mixed]}",
		"plan_policy.chain_policies: added OSMOSIS{requirements: [rest/GET ext=[archive] mixed, tendermintrpc ext=[archive] mixed, grpc ext=[archive] mixed]}",
		"plan_policy.chain_policies: added COSMOSHUB{requirements: [rest/GET ext=[archive] mixed, tendermintrpc ext=[archive] mixed, grpc ext=[archive] mixed]}",
		"plan_policy.chain_policies: added SQDSUBGRAPH{requirements: [rest/POST addon=compound-v3 mixed, rest/POST addon=aave-v3 mixed]}",
		"plan_policy.chain_policies: added *{}",
	}
	require.Equal(t, expected, types.DiffPlans(original.Proposal.Plans[0], updated.Proposal.Plans[0]))
}
