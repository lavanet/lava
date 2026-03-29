package types

import "fmt"

// CurrentContainer holds the index of a Combinable within its parent slice so
// that overwrite operations can update the correct position in-place.
type CurrentContainer struct {
	index             int
	currentCombinable Combinable
}

// Combinable is implemented by all types that can be merged across ApiCollection
// inheritance boundaries (Api, Header, ParseDirective, Extension, Verification).
type Combinable interface {
	// Differeniator returns the unique key used to identify this item within
	// its containing collection (e.g. Api.Name, FUNCTION_TAG string, etc.).
	// The intentional spelling matches the original codebase.
	Differeniator() string

	// GetEnabled reports whether this item is active.
	GetEnabled() bool

	// Equal reports deep equality against another Combinable of the same type.
	Equal(interface{}) bool

	// Overwrite is called when this Combinable is the survivor of a conflict
	// with other. It may mutate the receiver to merge data from other and
	// returns (overwritten, true) when a merge was performed, or (self, false)
	// when no special handling is needed.
	Overwrite(other Combinable) (overwritten Combinable, isOverwritten bool)
}

// ---------------------------------------------------------------------------
// Differeniator implementations
// ---------------------------------------------------------------------------

func (api *Api) Differeniator() string { return api.Name }

func (h *Header) Differeniator() string { return h.Name }

func (parsing *ParseDirective) Differeniator() string {
	// Subscribe and Unsubscribe tags may appear multiple times (e.g. Solana),
	// so include the ApiName to keep them distinct.
	if parsing.FunctionTag == FUNCTION_TAG_SUBSCRIBE || parsing.FunctionTag == FUNCTION_TAG_UNSUBSCRIBE {
		return parsing.FunctionTag.String() + "_" + parsing.ApiName
	}
	return parsing.FunctionTag.String()
}

func (e *Extension) Differeniator() string { return e.Name }

func (v *Verification) Differeniator() string { return v.Name }

// ---------------------------------------------------------------------------
// GetEnabled implementations (Api already has one as a getter)
// ---------------------------------------------------------------------------

func (h *Header) GetEnabled() bool       { return true }
func (p *ParseDirective) GetEnabled() bool { return true }
func (e *Extension) GetEnabled() bool    { return true }
func (v *Verification) GetEnabled() bool { return true }

// ---------------------------------------------------------------------------
// Overwrite implementations
// ---------------------------------------------------------------------------

func (api *Api) Overwrite(_ Combinable) (Combinable, bool) { return api, false }

func (h *Header) Overwrite(_ Combinable) (Combinable, bool) { return h, false }

func (parsing *ParseDirective) Overwrite(_ Combinable) (Combinable, bool) { return parsing, false }

func (e *Extension) Overwrite(_ Combinable) (Combinable, bool) { return e, false }

// Verification.Overwrite handles the case where this verification has no
// ParseDirective but the other one does: it borrows the directive and merges
// any non-overridden Values from the other verification.
func (v *Verification) Overwrite(other Combinable) (Combinable, bool) {
	if v.ParseDirective == nil {
		if otherV, ok := other.(*Verification); ok && otherV.ParseDirective != nil {
			v.ParseDirective = otherV.ParseDirective

			// Index existing values by extension to avoid duplicates.
			existing := make(map[string]struct{}, len(v.Values))
			for _, val := range v.Values {
				existing[val.Extension] = struct{}{}
			}
			for _, otherVal := range otherV.Values {
				if _, found := existing[otherVal.Extension]; !found {
					v.Values = append(v.Values, otherVal)
				}
			}
			return v, true
		}
	}
	return v, false
}

// ---------------------------------------------------------------------------
// Equal implementations
// ---------------------------------------------------------------------------

func (api *Api) Equal(that interface{}) bool {
	if that == nil {
		return api == nil
	}
	var other *Api
	switch t := that.(type) {
	case *Api:
		other = t
	case Api:
		other = &t
	default:
		return false
	}
	if other == nil {
		return api == nil
	}
	if api == nil {
		return false
	}
	return api.Enabled == other.Enabled &&
		api.Name == other.Name &&
		api.ComputeUnits == other.ComputeUnits &&
		api.ExtraComputeUnits == other.ExtraComputeUnits
}

func (h *Header) Equal(that interface{}) bool {
	if that == nil {
		return h == nil
	}
	var other *Header
	switch t := that.(type) {
	case *Header:
		other = t
	case Header:
		other = &t
	default:
		return false
	}
	if other == nil {
		return h == nil
	}
	if h == nil {
		return false
	}
	return h.Name == other.Name && h.Kind == other.Kind && h.FunctionTag == other.FunctionTag && h.Value == other.Value
}

func (parsing *ParseDirective) Equal(that interface{}) bool {
	if that == nil {
		return parsing == nil
	}
	var other *ParseDirective
	switch t := that.(type) {
	case *ParseDirective:
		other = t
	case ParseDirective:
		other = &t
	default:
		return false
	}
	if other == nil {
		return parsing == nil
	}
	if parsing == nil {
		return false
	}
	return parsing.FunctionTag == other.FunctionTag &&
		parsing.FunctionTemplate == other.FunctionTemplate &&
		parsing.ApiName == other.ApiName
}

func (e *Extension) Equal(that interface{}) bool {
	if that == nil {
		return e == nil
	}
	var other *Extension
	switch t := that.(type) {
	case *Extension:
		other = t
	case Extension:
		other = &t
	default:
		return false
	}
	if other == nil {
		return e == nil
	}
	if e == nil {
		return false
	}
	return e.Name == other.Name && e.CuMultiplier == other.CuMultiplier
}

func (v *Verification) Equal(that interface{}) bool {
	if that == nil {
		return v == nil
	}
	var other *Verification
	switch t := that.(type) {
	case *Verification:
		other = t
	case Verification:
		other = &t
	default:
		return false
	}
	if other == nil {
		return v == nil
	}
	if v == nil {
		return false
	}
	return v.Name == other.Name
}

// ---------------------------------------------------------------------------
// Generic combination helpers
// ---------------------------------------------------------------------------

// GetCurrentFromCombinable builds a map from differentiator key to
// CurrentContainer for all elements already present in a slice.
func GetCurrentFromCombinable[T Combinable](current []T) map[string]CurrentContainer {
	m := make(map[string]CurrentContainer, len(current))
	for idx, c := range current {
		m[c.Differeniator()] = CurrentContainer{index: idx, currentCombinable: c}
	}
	return m
}

// CombineFields merges parentCombinables into the accumulated mergedCombinables
// list, honouring allowOverwrite semantics.  Returns the updated list and map.
func CombineFields[T Combinable](
	childCombinables map[string]CurrentContainer,
	parentCombinables []T,
	mergedMap map[string]interface{},
	mergedCombinables []T,
	allowOverwrite bool,
) ([]T, map[string]interface{}, error) {
	for _, c := range parentCombinables {
		if !c.GetEnabled() {
			continue
		}
		key := c.Differeniator()
		if existing, ok := mergedMap[key]; ok {
			if !c.Equal(existing) {
				if !allowOverwrite {
					return nil, nil, fmt.Errorf("try overwrite existing collection combination %s", key)
				}
				if _, found := childCombinables[key]; !found {
					return nil, nil, fmt.Errorf("duplicate imported combinable: %s", key)
				}
			}
		}
		mergedMap[key] = c
		mergedCombinables = append(mergedCombinables, c)
	}
	return mergedCombinables, mergedMap, nil
}

// CombineUnique appends elements from appendFrom into appendTo, skipping items
// whose key already exists in currentMap.  When allowOverwrite is true,
// existing elements may be updated via their Overwrite method.
func CombineUnique[T Combinable](
	appendFrom, appendTo []T,
	currentMap map[string]CurrentContainer,
	allowOverwrite bool,
) ([]T, error) {
	for _, c := range appendFrom {
		current, found := currentMap[c.Differeniator()]
		if !found {
			appendTo = append(appendTo, c)
			continue
		}
		if !allowOverwrite {
			if !c.Equal(current.currentCombinable) {
				return nil, fmt.Errorf("existing combinable in collection combination %s", c.Differeniator())
			}
			continue
		}
		// Attempt an Overwrite to allow special merge behaviour (e.g. Verification).
		overwritten, isOverwritten := current.currentCombinable.Overwrite(c)
		if isOverwritten {
			if current.index >= len(appendTo) || appendTo[current.index].Differeniator() != c.Differeniator() {
				return nil, fmt.Errorf("differentiator mismatch in overwrite %s vs %s",
					c.Differeniator(), appendTo[current.index].Differeniator())
			}
			overwrittenT, ok := overwritten.(T)
			if !ok {
				return nil, fmt.Errorf("failed casting overwritten to T %s", overwritten.Differeniator())
			}
			appendTo[current.index] = overwrittenT
		}
	}
	return appendTo, nil
}
