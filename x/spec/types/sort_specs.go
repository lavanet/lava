package types

import (
	fmt "fmt"
)

// sortSpecsByHierarchy sorts specs based on their import dependencies
// Returns a sorted list of specs where specs with no imports come first,
// followed by specs that import from already processed specs
func SortSpecsByHierarchy(specs []Spec) ([]Spec, error) {
	// Create a list to store sorted specs
	sorted := make([]Spec, 0, len(specs))

	// Create a set to track processed specs
	processed := make(map[string]bool)
	// Create a set to track all specs
	specsExists := make(map[string]bool)
	for _, spec := range specs {
		specsExists[spec.Index] = true
	}

	// Helper function to check if all imports are processed
	canProcess := func(imports []string) bool {
		for _, imp := range imports {
			if !processed[imp] && specsExists[imp] {
				return false
			}
		}
		return true
	}

	// Keep processing until all specs are sorted
	for len(sorted) < len(specs) {
		progress := false

		for _, spec := range specs {
			// Skip if already processed
			if processed[spec.Index] {
				continue
			}

			// If spec has no imports or all its imports are processed
			if len(spec.Imports) == 0 || canProcess(spec.Imports) {
				sorted = append(sorted, spec)
				processed[spec.Index] = true
				progress = true
			}
		}

		// If no progress was made in this iteration, we have a circular dependency
		if !progress {
			return nil, fmt.Errorf("circular dependency detected in specs imports")
		}
	}

	return sorted, nil
}
