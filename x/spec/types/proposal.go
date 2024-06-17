package types

import (
	fmt "fmt"
	"strings"

	sdkerrors "cosmossdk.io/errors"
)

func (spec Spec) ValidateBasic() error {
	if len(strings.TrimSpace(spec.Name)) == 0 {
		return sdkerrors.Wrapf(ErrBlankSpecName, "spec name cannot be blank %v", spec)
	}
	if len(strings.TrimSpace(spec.Index)) == 0 {
		return sdkerrors.Wrap(ErrBlankSpecName, "spec index cannot be blank")
	}
	if len(spec.ApiCollections) == 0 && len(spec.Imports) == 0 {
		return sdkerrors.Wrap(ErrEmptyApis, "api list cannot be empty")
	}

	for _, apiCollection := range spec.ApiCollections {
		checkUnique := map[string]bool{}
		for idx, api := range apiCollection.Apis {
			if len(strings.TrimSpace(api.Name)) == 0 {
				prevApi := ""
				if idx != 0 {
					prevApi = apiCollection.Apis[idx-1].Name
				} else if idx+1 < len(apiCollection.Apis) {
					prevApi = apiCollection.Apis[idx+1].Name
				}
				return sdkerrors.Wrapf(ErrBlankApiName, "api name cannot be blank, %#v, in spec %s:%s, previous/next api: %s", apiCollection.CollectionData, spec.Index, spec.Name, prevApi)
			}
			if _, ok := checkUnique[api.Name]; ok {
				return sdkerrors.Wrap(ErrDuplicateApiName, fmt.Sprintf("api name must be unique: %s", api.Name))
			}
			checkUnique[api.Name] = true
		}
	}
	return nil
}

func stringSpec(spec Spec, b strings.Builder) strings.Builder {
	b.WriteString(fmt.Sprintf(`    Spec name:
	Name: %s, Spec index: %s, Enabled: %t, ApiCollections: %d
`, spec.Name, spec.Index, spec.Enabled, len(spec.ApiCollections)))
	for _, collection := range spec.ApiCollections {
		b.WriteString(fmt.Sprintf(`        ApiCollection: %v
		`, collection.CollectionData))
		for _, api := range collection.Apis {
			b.WriteString(fmt.Sprintf(`        Api:
				Name: %s, Enabled: %t, ComputeUntis: %d
			`, api.Name, api.Enabled, api.ComputeUnits))
		}
	}
	return b
}
