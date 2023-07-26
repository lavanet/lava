package types

import fmt "fmt"

type Combinable interface {
	Differeniator() string
	GetEnabled() bool
	Equal(interface{}) bool
}

func (api *Api) Differeniator() string {
	return api.Name
}

func (h *Header) Differeniator() string {
	return h.Name
}

func (parsing *ParseDirective) Differeniator() string {
	return parsing.FunctionTag.String()
}

func (e *Extension) Differeniator() string {
	return e.Name
}

func (v *Verification) Differeniator() string {
	return v.Name
}

// Api has GetEnabled

func (h *Header) GetEnabled() bool {
	return true
}

func (h *ParseDirective) GetEnabled() bool {
	return true
}

func (h *Extension) GetEnabled() bool {
	return true
}

func (h *Verification) GetEnabled() bool {
	return true
}

// this function is used to combine combinables from the parents and the current collection and combine them into the current collection
// it takes a map of current existing combinables, the parent combinables in mergedMap, the list of combinables from the parents, and the list of combinables in the current spec
// toCombine and currentMap hold the same data arranged in a different set, the same for mergedMap and combineFrom
func CombineFields[T Combinable](childCombinables map[string]interface{}, parentCombinables []T, mergedMap map[string]interface{}, mergedCombinables []T, allowOverwrite bool) ([]T, map[string]interface{}, error) {
	for _, combinable := range parentCombinables {
		if !combinable.GetEnabled() {
			continue
		}
		if existing, ok := mergedMap[combinable.Differeniator()]; ok {
			// only matters if they are not the same
			if !combinable.Equal(existing) {
				// was already existing in mergedApis
				if !allowOverwrite {
					// if we don't allow overwriting by the calling collection, then a duplication is an error, but only if it's not equal
					return nil, nil, fmt.Errorf("try overwrite existing collection combination %s", combinable.Differeniator())
				}
				if _, found := childCombinables[combinable.Differeniator()]; !found {
					return nil, nil, fmt.Errorf("duplicate imported combinable: %s", combinable.Differeniator())
				}
			}
		}
		mergedMap[combinable.Differeniator()] = combinable
		mergedCombinables = append(mergedCombinables, combinable)
	}
	return mergedCombinables, mergedMap, nil
}

func GetCurrentFromCombinable[T Combinable](current []T) (currentMap map[string]interface{}) {
	currentMap = map[string]interface{}{}
	for _, combinable := range current {
		currentMap[combinable.Differeniator()] = combinable
	}
	return currentMap
}

func CombineUnique[T Combinable](appendFrom []T, appendTo []T, currentMap map[string]interface{}, allowOverwrite bool) ([]T, error) {
	for _, combinable := range appendFrom {
		if current, found := currentMap[combinable.Differeniator()]; !found {
			appendTo = append(appendTo, combinable)
		} else if !allowOverwrite {
			// if they are the same it's allowed
			if !combinable.Equal(current) {
				return nil, fmt.Errorf("existing combinable in collection combination %s ", combinable.Differeniator())
			}
		}
	}
	return appendTo, nil
}
