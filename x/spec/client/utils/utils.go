package utils

import (
	"os"
	"strings"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/x/spec/types"
)

type (
	ApiInterfaceJSON struct {
		Interface         string `json:"interface" yaml:"interface"`
		Type              string `json:"type" yaml:"type"`
		ExtraComputeUnits uint   `json:"extra_compute_units" yaml:"extra_compute_units"`
	}

	ApiJSON struct {
		Name          string              `json:"name" yaml:"name"`
		ComputeUnits  uint                `json:"compute_units" yaml:"compute_units"`
		Enabled       bool                `json:"enabled" yaml:"enabled"`
		ApiInterfaces []ApiInterfaceJSON  `json:"apiInterfaces" yaml:"apiInterfaces"`
		BlockParsing  types.BlockParser   `json:"block_parsing" yaml:"block_parsing"`
		Category      *types.SpecCategory `json:"category"`
		Parsing       types.Parsing       `json:"parsing" yaml:"parsing"`
	}

	SpecJSON struct {
		ChainID string    `json:"chainid" yaml:"chainid"`
		Name    string    `json:"name" yaml:"name"`
		Enabled bool      `json:"enabled" yaml:"enabled"`
		Apis    []ApiJSON `json:"apis" yaml:"apis"`

		ReliabilityThreshold      uint32 `json:"reliability_threshold" yaml:"enabled"`
		ComparesHashes            bool   `json:"compares_hashes" yaml:"enabled"`
		FinalizationCriteria      uint32 `json:"finalization_criteria" yaml:"finalization_criteria"`
		SavedBlocks               uint32 `json:"saved_blocks" yaml:"saved_blocks"`
		AverageBlockTime          int64  `json:"average_block_time" yaml:"enabled"`
		AllowedBlockLagForQosSync int64  `json:"allowed_block_lag_for_qos_sync" yaml:"enabled"`
	}

	SpecAddProposalJSON struct {
		Title       string     `json:"title" yaml:"title"`
		Description string     `json:"description" yaml:"description"`
		Specs       []SpecJSON `json:"specs" yaml:"changes"`
		Deposit     string     `json:"deposit" yaml:"deposit"`
	}
)

// Get specs in form
func (pcj SpecAddProposalJSON) ToSpecs() []types.Spec {
	ret := []types.Spec{}
	for _, spec := range pcj.Specs {
		apis := []types.ServiceApi{}
		for _, api := range spec.Apis {
			apis = append(apis, types.ServiceApi{
				Name:          api.Name,
				ComputeUnits:  uint64(api.ComputeUnits),
				Enabled:       api.Enabled,
				ApiInterfaces: ConvertJSONApiInterface(api.ApiInterfaces),
				BlockParsing:  api.BlockParsing,
				Category:      api.Category,
				Parsing:       api.Parsing,
			})
		}
		ret = append(ret, types.Spec{
			Index:                     spec.ChainID,
			Name:                      spec.Name,
			Enabled:                   spec.Enabled,
			Apis:                      apis,
			ReliabilityThreshold:      spec.ReliabilityThreshold,
			ComparesHashes:            spec.ComparesHashes,
			FinalizationCriteria:      spec.FinalizationCriteria,
			SavedBlocks:               spec.SavedBlocks,
			AverageBlockTime:          spec.AverageBlockTime,
			AllowedBlockLagForQosSync: spec.AllowedBlockLagForQosSync,
		})
	}
	return ret
}

// Parse spec add proposal JSON form file
func ParseSpecAddProposalJSON(cdc *codec.LegacyAmino, proposalFile string) (ret SpecAddProposalJSON, err error) {
	for _, fileName := range strings.Split(proposalFile, ",") {
		proposal := SpecAddProposalJSON{}

		contents, err := os.ReadFile(fileName)
		if err != nil {
			return proposal, err
		}

		if err := cdc.UnmarshalJSON(contents, &proposal); err != nil {
			return proposal, err
		}
		if len(ret.Specs) > 0 {
			ret.Specs = append(ret.Specs, proposal.Specs...)
			ret.Description = proposal.Description + " " + ret.Description
			ret.Title = proposal.Title + " " + ret.Title
			retDeposit, err := sdk.ParseCoinNormalized(ret.Deposit)
			if err != nil {
				return proposal, err
			}
			proposalDeposit, err := sdk.ParseCoinNormalized(proposal.Deposit)
			if err != nil {
				return proposal, err
			}
			ret.Deposit = retDeposit.Add(proposalDeposit).String()
		} else {
			ret = proposal
		}
	}
	return ret, nil
}

func ConvertJSONApiInterface(apiinterfacesJSON []ApiInterfaceJSON) (apiInterfaces []types.ApiInterface) {
	for _, apiinterface := range apiinterfacesJSON {
		apiInterfaces = append(apiInterfaces, types.ApiInterface{Interface: apiinterface.Interface, Type: apiinterface.Type, ExtraComputeUnits: uint64(apiinterface.ExtraComputeUnits)})
	}

	return
}
