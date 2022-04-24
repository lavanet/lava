package utils

import (
	"os"

	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/lavanet/lava/x/spec/types"
)

type (
	ApiInterfaceJSON struct {
		Interface string `json:"interface" yaml:"interface"`
		Type      string `json:"type" yaml:"type"`
	}

	ApiJSON struct {
		Name          string             `json:"name" yaml:"name"`
		ComputeUnits  uint               `json:"compute_units" yaml:"compute_units"`
		Status        bool               `json:"status" yaml:"status"`
		ApiInterfaces []ApiInterfaceJSON `json:"apiInterfaces" yaml:"apiInterfaces"`
	}

	SpecJSON struct {
		Name   string    `json:"name" yaml:"name"`
		Status bool      `json:"status" yaml:"status"`
		Apis   []ApiJSON `json:"apis" yaml:"apis"`
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
				Status:        api.Status,
				ApiInterfaces: ConvertJSONApiInterface(api.ApiInterfaces),
			})

		}

		ret = append(ret, types.Spec{
			Name:   spec.Name,
			Status: spec.Status,
			Apis:   apis,
		})
	}
	return ret
}

// Parse spec add proposal JSON form file
func ParseSpecAddProposalJSON(cdc *codec.LegacyAmino, proposalFile string) (SpecAddProposalJSON, error) {
	proposal := SpecAddProposalJSON{}

	contents, err := os.ReadFile(proposalFile)
	if err != nil {
		return proposal, err
	}

	if err := cdc.UnmarshalJSON(contents, &proposal); err != nil {
		return proposal, err
	}

	return proposal, nil
}

func ConvertJSONApiInterface(apiinterfacesJSON []ApiInterfaceJSON) (ApiInterfaces []types.ApiInterface) {

	for _, apiinterface := range apiinterfacesJSON {
		ApiInterfaces = append(ApiInterfaces, types.ApiInterface{Interface: apiinterface.Interface, Type: apiinterface.Type})
	}

	return
}
