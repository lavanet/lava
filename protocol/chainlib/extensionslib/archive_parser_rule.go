package extensionslib

import (
	"github.com/lavanet/lava/v5/utils"
	spectypes "github.com/lavanet/lava/v5/x/spec/types"
)

type ArchiveParserRule struct {
	extension *spectypes.Extension
}

func (apr ArchiveParserRule) isPassingRule(extensionChainMessage ExtensionsChainMessage, latestBlock uint64) bool {
	_, earliestRequestedBlock := extensionChainMessage.RequestedBlock()
	ruleBlock := uint64(0)
	if apr.extension.Rule != nil {
		ruleBlock = apr.extension.Rule.Block
	}
	utils.LavaFormatTrace("[Archive Debug] ArchiveParserRule.isPassingRule called",
		utils.LogAttr("earliestRequestedBlock", earliestRequestedBlock),
		utils.LogAttr("latestBlock", latestBlock),
		utils.LogAttr("ruleBlock", ruleBlock))

	if earliestRequestedBlock < 0 {
		// if asking for the latest block, or an api that doesn't have a specific block requested then it's not archive
		isEarliest := earliestRequestedBlock == spectypes.EARLIEST_BLOCK
		utils.LavaFormatTrace("[Archive Debug] Negative block check",
			utils.LogAttr("earliestRequestedBlock", earliestRequestedBlock),
			utils.LogAttr("isEarliestBlock", isEarliest),
			utils.LogAttr("archiveDecision", isEarliest))
		return isEarliest // only earliest should go to archive
	}

	if latestBlock == 0 {
		utils.LavaFormatTrace("[Archive Debug] Latest block is 0, forcing archive", utils.LogAttr("archiveDecision", true))
		return true
	}

	if uint64(earliestRequestedBlock) >= latestBlock {
		utils.LavaFormatTrace("[Archive Debug] Requested block >= latest block, no archive needed",
			utils.LogAttr("earliestRequestedBlock", earliestRequestedBlock),
			utils.LogAttr("latestBlock", latestBlock),
			utils.LogAttr("archiveDecision", false))
		return false
	}

	if apr.extension.Rule != nil && apr.extension.Rule.Block != 0 {
		// Prevent underflow: if latestBlock <= ruleBlock, no blocks are old enough
		if latestBlock <= apr.extension.Rule.Block {
			utils.LavaFormatTrace("[Archive Debug] Latest block <= rule block, no archive needed",
				utils.LogAttr("latestBlock", latestBlock),
				utils.LogAttr("ruleBlock", apr.extension.Rule.Block),
				utils.LogAttr("archiveDecision", false))
			return false
		}
		threshold := latestBlock - apr.extension.Rule.Block
		isOldEnough := uint64(earliestRequestedBlock) < threshold
		utils.LavaFormatTrace("[Archive Debug] Checking configured block threshold",
			utils.LogAttr("ruleBlock", apr.extension.Rule.Block),
			utils.LogAttr("threshold", threshold),
			utils.LogAttr("earliestRequestedBlock", earliestRequestedBlock),
			utils.LogAttr("isOldEnough", isOldEnough),
			utils.LogAttr("archiveDecision", isOldEnough))
		if isOldEnough {
			return true
		}
	} else {
		ruleBlock := uint64(0)
		if apr.extension.Rule != nil {
			ruleBlock = apr.extension.Rule.Block
		}
		utils.LavaFormatTrace("[Archive Debug] No rule configured or rule block is 0, no archive",
			utils.LogAttr("hasRule", apr.extension.Rule != nil),
			utils.LogAttr("ruleBlock", ruleBlock),
			utils.LogAttr("archiveDecision", false))
	}

	utils.LavaFormatTrace("[Archive Debug] Archive rule not triggered, returning false",
		utils.LogAttr("archiveDecision", false))
	return false
}
