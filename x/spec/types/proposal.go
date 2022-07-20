package types

import (
	fmt "fmt"
	"strings"

	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

func checkSpecProposal(spec Spec) error {
	if len(strings.TrimSpace(spec.Name)) == 0 {
		return sdkerrors.Wrap(ErrBlankSpecName, "spec name cannot be blank")
	}
	if len(strings.TrimSpace(spec.Index)) == 0 {
		return sdkerrors.Wrap(ErrBlankSpecName, "spec index cannot be blank")
	}
	if len(spec.Apis) == 0 {
		return sdkerrors.Wrap(ErrEmptyApis, "api list cannot be empty")
	}

	checkUnique := map[string]bool{}
	for _, api := range spec.Apis {
		if len(strings.TrimSpace(api.Name)) == 0 {
			return sdkerrors.Wrap(ErrBlankApiName, "api name cannot be blank")
		}
		if _, ok := checkUnique[api.Name]; ok {
			return sdkerrors.Wrap(ErrDuplicateApiName, fmt.Sprintf("api name must be unique: %s", api.Name))
		}
		if len(api.ApiInterfaces) == 0 {
			return sdkerrors.Wrap(ErrDuplicateApiName, "api interface cannot be empty")
		}
		checkUnique[api.Name] = true
	}
	return nil
}

func stringSpec(spec Spec, b strings.Builder) strings.Builder {

	b.WriteString(fmt.Sprintf(`    Spec name:
	Name: %s, Spec index: %s, Enabled: %s, Apis: %d
`, spec.Name, spec.Index, spec.Enabled, len(spec.Apis)))

	for _, api := range spec.Apis {
		b.WriteString(fmt.Sprintf(`        Api:
		      Name: %s, Enabled: %s, ComputeUntis: %d
		`, api.Name, api.Enabled, api.ComputeUnits))
	}

	return b
}
