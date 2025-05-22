package types

import (
	fmt "fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

// Define a type for the GetSpec function
type GetSpecFunc func(ctx sdk.Context, index string) (Spec, bool)

// doExpandSpec performs the actual work and recusion for ExpandSpec above.
func DoExpandSpec(
	ctx sdk.Context,
	spec *Spec,
	depends map[string]bool,
	inherit *map[string]bool,
	details string,
	getSpecFn GetSpecFunc,
) (string, error) {
	if spec == nil {
		return "", fmt.Errorf("doExpandSpec: spec is nil")
	}
	parentsCollections := map[CollectionData][]*ApiCollection{}

	if len(spec.Imports) != 0 {
		var parents []Spec

		// update (cumulative) inherit
		for _, index := range spec.Imports {
			(*inherit)[index] = true
		}

		// visual markers when import deepens
		details += "->["

		// recursion to get all parent specs (DFS)
		comma := ""
		for _, index := range spec.Imports {
			imported, found := getSpecFn(ctx, index)
			// import of unknown Spec not allowed
			if !found {
				details += fmt.Sprintf("%s%s(unknown)", comma, index)
				return details, fmt.Errorf("imported spec unknown: %s", index)
			}

			details += fmt.Sprintf("%s%s", comma, index)

			// loop in the recursion not allowed
			if _, found := depends[index]; found {
				return details, fmt.Errorf("import loops not allowed for spec: %s", index)
			}

			depends[index] = true
			details, err := DoExpandSpec(ctx, &imported, depends, inherit, details, getSpecFn)
			if err != nil {
				return details, err
			}
			delete(depends, index)

			parents = append(parents, imported)
			comma = ","
		}

		details += "]"

		for _, parent := range parents {
			for _, parentCollection := range parent.ApiCollections {
				// ignore disabled apiCollections
				if !parentCollection.Enabled {
					continue
				}
				if parentsCollections[parentCollection.CollectionData] == nil {
					parentsCollections[parentCollection.CollectionData] = []*ApiCollection{}
				}
				parentsCollections[parentCollection.CollectionData] = append(parentsCollections[parentCollection.CollectionData], parentCollection)
			}
		}
	}

	myCollections := map[CollectionData]*ApiCollection{}
	for _, collection := range spec.ApiCollections {
		myCollections[collection.CollectionData] = collection
	}

	for _, collection := range spec.ApiCollections {
		err := collection.InheritAllFields(myCollections, parentsCollections[collection.CollectionData])
		if err != nil {
			return details, err
		}
		delete(parentsCollections, collection.CollectionData)
	}

	// combine left over apis not overwritten by current spec
	err := spec.CombineCollections(parentsCollections)
	if err != nil {
		return details, err
	}

	return details, nil
}
