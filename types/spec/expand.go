package spec

import (
	"context"
	"fmt"
)

// GetSpecFunc is the callback signature used by DoExpandSpec to retrieve a
// parent spec by its chain index.  The context parameter is accepted for
// interface compatibility with callers that hold a Cosmos sdk.Context (which
// satisfies context.Context); it is not used internally.
type GetSpecFunc func(ctx context.Context, index string) (Spec, bool)

// DoExpandSpec recursively resolves all Imports for spec, merging the API
// collections of each imported parent into spec.  depends tracks the set of
// spec indices currently on the call stack to detect import cycles; inherit
// accumulates every transitively imported index.  details is a human-readable
// trace string that is extended with each recursive call and returned for
// diagnostic purposes.
func DoExpandSpec(
	ctx context.Context,
	spec *Spec,
	depends map[string]bool,
	inherit *map[string]bool,
	details string,
	getSpecFn GetSpecFunc,
) (string, error) {
	if spec == nil {
		return "", fmt.Errorf("DoExpandSpec: spec is nil")
	}

	parentsCollections := map[CollectionData][]*ApiCollection{}

	if len(spec.Imports) != 0 {
		// Accumulate every imported index in the inherit set.
		for _, index := range spec.Imports {
			(*inherit)[index] = true
		}

		details += "->["

		var parents []Spec
		comma := ""

		for _, index := range spec.Imports {
			imported, found := getSpecFn(ctx, index)
			if !found {
				details += fmt.Sprintf("%s%s(unknown)", comma, index)
				return details, fmt.Errorf("imported spec unknown: %s", index)
			}

			details += fmt.Sprintf("%s%s", comma, index)

			if _, found := depends[index]; found {
				return details, fmt.Errorf("import loops not allowed for spec: %s", index)
			}

			depends[index] = true
			var err error
			details, err = DoExpandSpec(ctx, &imported, depends, inherit, details, getSpecFn)
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
				if !parentCollection.Enabled {
					continue
				}
				key := parentCollection.CollectionData
				if parentsCollections[key] == nil {
					parentsCollections[key] = []*ApiCollection{}
				}
				parentsCollections[key] = append(parentsCollections[key], parentCollection)
			}
		}
	}

	myCollections := make(map[CollectionData]*ApiCollection, len(spec.ApiCollections))
	for _, collection := range spec.ApiCollections {
		myCollections[collection.CollectionData] = collection
	}

	for _, collection := range spec.ApiCollections {
		if err := collection.InheritAllFields(myCollections, parentsCollections[collection.CollectionData]); err != nil {
			return details, err
		}
		delete(parentsCollections, collection.CollectionData)
	}

	if err := spec.CombineCollections(parentsCollections); err != nil {
		return details, err
	}

	return details, nil
}
