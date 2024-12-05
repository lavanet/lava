package types

import (
	"fmt"
)

// this means the current collection data can be expanded from other, i.e other is allowed to be in InheritanceApis
func (cd *CollectionData) CanExpand(other *CollectionData) bool {
	return cd.ApiInterface == other.ApiInterface && cd.Type == other.Type || other.ApiInterface == ""
}

// expand is called within the same spec apiCollections, to manage inheritance within collections of different add_ons
func (apic *ApiCollection) Expand(myCollections map[CollectionData]*ApiCollection, dependencies map[CollectionData]struct{}) error {
	dependencies[apic.CollectionData] = struct{}{}
	defer delete(dependencies, apic.CollectionData)
	inheritanceApis := apic.InheritanceApis
	apic.InheritanceApis = []*CollectionData{} // delete inheritance so if someone calls expand on this in the future without dependency we don't repeat this
	relevantCollections := []*ApiCollection{}
	for _, inheritingCollection := range inheritanceApis {
		if collection, ok := myCollections[*inheritingCollection]; ok {
			if !apic.CollectionData.CanExpand(&collection.CollectionData) {
				return fmt.Errorf("invalid inheriting collection %v", inheritingCollection)
			}
			if _, ok := dependencies[collection.CollectionData]; ok {
				return fmt.Errorf("circular dependency in inheritance, %v", collection)
			}
			err := collection.Expand(myCollections, dependencies)
			if err != nil {
				return err
			}
			relevantCollections = append(relevantCollections, collection)
		} else {
			return fmt.Errorf("did not find inheritingCollection in myCollections %v", inheritingCollection)
		}
	}
	// since expand is called within the same spec it needs to combine with disabled apiCollections
	return apic.CombineWithOthers(relevantCollections, true, true)
}

// inherit is
func (apic *ApiCollection) Inherit(relevantCollections []*ApiCollection, dependencies map[CollectionData]struct{}) error {
	// do not set dependencies because this mechanism protects inheritance within the same spec and inherit is inheritance between different specs so same type is allowed
	return apic.CombineWithOthers(relevantCollections, false, true)
}

func (apic *ApiCollection) Equals(other *ApiCollection) bool {
	return other.CollectionData == apic.CollectionData
}

// assumes relevantParentCollections are already expanded
func (apic *ApiCollection) InheritAllFields(myCollections map[CollectionData]*ApiCollection, relevantParentCollections []*ApiCollection) error {
	for _, other := range relevantParentCollections {
		if !apic.Equals(other) {
			return fmt.Errorf("incompatible inheritance, apiCollections aren't equal %v", apic)
		}
	}
	err := apic.Expand(myCollections, map[CollectionData]struct{}{})
	if err != nil {
		return err
	}
	return apic.Inherit(relevantParentCollections, map[CollectionData]struct{}{})
}

// this function combines apis, headers and parsers into the api collection from others. it does not check type compatibility
// changes in place inside the apic
// nil merge maps means not to combine that field
func (apic *ApiCollection) CombineWithOthers(others []*ApiCollection, combineWithDisabled, allowOverwrite bool) (err error) {
	if apic == nil {
		return fmt.Errorf("CombineWithOthers: API collection is nil")
	}
	mergedApis := map[string]interface{}{}
	mergedHeaders := map[string]interface{}{}
	mergedParsers := map[string]interface{}{}
	mergedExtensions := map[string]interface{}{}
	mergedVerifications := map[string]interface{}{}

	mergedApisList := []*Api{}
	mergedHeadersList := []*Header{}
	mergedParsersList := []*ParseDirective{}
	mergedExtensionsList := []*Extension{}
	mergedVerificationsList := []*Verification{}

	currentApis := GetCurrentFromCombinable(apic.Apis)
	currentHeaders := GetCurrentFromCombinable(apic.Headers)
	currentParsers := GetCurrentFromCombinable(apic.ParseDirectives)
	currentExtensions := GetCurrentFromCombinable(apic.Extensions)
	currentVerifications := GetCurrentFromCombinable(apic.Verifications)

	for _, collection := range others {
		if !collection.Enabled && !combineWithDisabled {
			continue
		}

		mergedApisList, mergedApis, err = CombineFields(currentApis, collection.Apis, mergedApis, mergedApisList, allowOverwrite)
		if err != nil {
			return fmt.Errorf("merging apis error %w, %v other collection %v", err, apic, collection.CollectionData)
		}

		mergedHeadersList, mergedHeaders, err = CombineFields(currentHeaders, collection.Headers, mergedHeaders, mergedHeadersList, allowOverwrite)
		if err != nil {
			return fmt.Errorf("merging headers error %w, %v other collection %v", err, apic, collection.CollectionData)
		}

		mergedParsersList, mergedParsers, err = CombineFields(currentParsers, collection.ParseDirectives, mergedParsers, mergedParsersList, allowOverwrite)
		if err != nil {
			return fmt.Errorf("merging parse directives error %w, %v other collection %v", err, apic, collection.CollectionData)
		}

		mergedExtensionsList, mergedExtensions, err = CombineFields(currentExtensions, collection.Extensions, mergedExtensions, mergedExtensionsList, allowOverwrite)
		if err != nil {
			return fmt.Errorf("merging extensions error %w, %v other collection %v", err, apic, collection.CollectionData)
		}

		mergedVerificationsList, mergedVerifications, err = CombineFields(currentVerifications, collection.Verifications, mergedVerifications, mergedVerificationsList, allowOverwrite)
		if err != nil {
			return fmt.Errorf("merging verifications error %w, %v other collection %v", err, apic, collection.CollectionData)
		}
	}

	// merge collected APIs into current apiCollection's APIs (unless overridden)
	apic.Apis, err = CombineUnique(mergedApisList, apic.Apis, currentApis, allowOverwrite)
	if err != nil {
		return fmt.Errorf("error %w in apis combination in collection %#v", err, apic)
	}

	// merge collected headers into current apiCollection's headers (unless overridden)
	apic.Headers, err = CombineUnique(mergedHeadersList, apic.Headers, currentHeaders, allowOverwrite)
	if err != nil {
		return fmt.Errorf("error %w in headers combination in collection %#v", err, apic)
	}
	// merge collected functionTags into current apiCollection's parsing (unless overridden)
	apic.ParseDirectives, err = CombineUnique(mergedParsersList, apic.ParseDirectives, currentParsers, allowOverwrite)
	if err != nil {
		return fmt.Errorf("error %w in parse directive combination in collection %#v", err, apic)
	}

	// merge collected extensions into current apiCollection's parsing (unless overridden)
	apic.Extensions, err = CombineUnique(mergedExtensionsList, apic.Extensions, currentExtensions, allowOverwrite)
	if err != nil {
		return fmt.Errorf("error %w in extension combination in collection %#v", err, apic)
	}

	// merge collected verifications into current apiCollection's parsing (unless overridden)
	apic.Verifications, err = CombineUnique(mergedVerificationsList, apic.Verifications, currentVerifications, allowOverwrite)
	if err != nil {
		return fmt.Errorf("error %w in verification combination in collection %#v", err, apic)
	}

	return nil
}

func (sc SpecCategory) Combine(other SpecCategory) SpecCategory {
	returnedCategory := SpecCategory{
		Deterministic: sc.Deterministic && other.Deterministic,
		Local:         sc.Local || other.Local,
		Subscription:  sc.Subscription || other.Subscription,
		Stateful:      sc.Stateful + other.Stateful,
		HangingApi:    sc.HangingApi || other.HangingApi,
	}
	return returnedCategory
}
