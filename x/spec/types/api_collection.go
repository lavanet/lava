package types

import (
	"fmt"
)

// this means the current collection data can be expanded from other, i.e other is allowed to be in InheritanceApis
func (cd *CollectionData) CanExpand(other *CollectionData) bool {
	return cd.ApiInterface == other.ApiInterface && cd.Type == other.Type && cd.InternalPath == other.InternalPath || other.ApiInterface == ""
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
	return apic.CombineWithOthers(relevantCollections, true, make(map[string]struct{}), make(map[string]struct{}), make(map[string]struct{}))
}

// inherit is
func (apic *ApiCollection) Inherit(relevantCollections []*ApiCollection, dependencies map[CollectionData]struct{}) error {
	// do not set dependencies because this mechanism protects inheritance within the same spec and inherit is inheritance between different specs so same type is allowed
	return apic.CombineWithOthers(relevantCollections, true, make(map[string]struct{}), make(map[string]struct{}), make(map[string]struct{}))
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
func (apic *ApiCollection) CombineWithOthers(others []*ApiCollection, allowOverwrite bool, mergedApis map[string]struct{}, mergedHeaders map[string]struct{}, mergedParsers map[string]struct{}) error {
	mergedHeadersList := []*Header{}
	mergedApisList := []*Api{}
	mergedParserList := []*Parsing{}
	currentApis := make(map[string]struct{}, 0)
	currentHeaders := make(map[string]struct{}, 0)
	currentParsers := make(map[string]struct{}, 0)
	if mergedApis != nil {
		for _, api := range apic.Apis {
			currentApis[api.Name] = struct{}{}
		}
	}
	if mergedHeaders != nil {
		for _, header := range apic.Headers {
			currentHeaders[header.Name] = struct{}{}
		}
	}
	if mergedParsers != nil {
		for _, parser := range apic.Parsing {
			currentParsers[parser.FunctionTag] = struct{}{}
		}
	}

	for _, collection := range others {
		if !collection.Enabled {
			continue
		}
		if mergedApis != nil {
			for _, api := range collection.Apis {
				if !api.Enabled {
					continue
				}
				if _, ok := mergedApis[api.Name]; ok {
					// was already existing in mergedApis
					if !allowOverwrite {
						// if we don't allow overwriting by the calling collection, then a duplication is an error
						return fmt.Errorf("existing api in collection combination %s %v other collection %v", api.Name, apic, collection.CollectionData)
					}
					if _, found := currentApis[api.Name]; !found {
						return fmt.Errorf("duplicate imported api: %s (in collection: %v)", api.Name, collection.CollectionData)
					}
				}
				mergedApis[api.Name] = struct{}{}
				mergedApisList = append(mergedApisList, api)
			}
		}
		if mergedHeaders != nil {
			for _, header := range collection.Headers {
				if _, ok := mergedHeaders[header.Name]; ok {
					// was already existing in mergedHeaders
					if !allowOverwrite {
						// if we don't allow overwriting by the calling collection, then a duplication is an error
						return fmt.Errorf("existing header in collection combination %s %v other collection %v", header.Name, apic, collection.CollectionData)
					}
					if _, found := currentHeaders[header.Name]; !found {
						return fmt.Errorf("duplicate imported header: %s (in collection: %v)", header.Name, collection.CollectionData)
					}
				}
				mergedHeaders[header.Name] = struct{}{}
				mergedHeadersList = append(mergedHeadersList, header)
			}
		}
		if mergedParsers != nil {
			for _, parsing := range collection.Parsing {
				if _, ok := mergedParsers[parsing.FunctionTag]; ok {
					// was already existing in mergedParsers
					if !allowOverwrite {
						// if we don't allow overwriting by the calling collection, then a duplication is an error
						return fmt.Errorf("existing parsing in collection combination %s %v other collection %v", parsing.FunctionTag, apic, collection.CollectionData)
					}
					if _, found := currentParsers[parsing.FunctionTag]; !found {
						return fmt.Errorf("duplicate imported parsing: %s (in collection: %v)", parsing.FunctionTag, collection.CollectionData)
					}
				}
				mergedParsers[parsing.FunctionTag] = struct{}{}
				mergedParserList = append(mergedParserList, parsing)
			}
		}
	}

	// merge collected APIs into current apiCollection's APIs (unless overridden)
	for _, api := range mergedApisList {
		if _, found := currentApis[api.Name]; !found {
			apic.Apis = append(apic.Apis, api)
		} else if !allowOverwrite {
			return fmt.Errorf("existing api in collection combination %s %v", api.Name, apic)
		}
	}

	// merge collected headers into current apiCollection's headers (unless overridden)
	for _, header := range mergedHeadersList {
		if _, found := currentHeaders[header.Name]; !found {
			apic.Headers = append(apic.Headers, header)
		} else if !allowOverwrite {
			return fmt.Errorf("existing header in collection combination %s %v", header.Name, apic)
		}
	}

	// merge collected functionTags into current apiCollection's parsing (unless overridden)
	for _, parser := range mergedParserList {
		if _, found := currentParsers[parser.FunctionTag]; !found {
			apic.Parsing = append(apic.Parsing, parser)
		} else if !allowOverwrite {
			return fmt.Errorf("existing api in collection combination %s %v", parser.FunctionTag, apic)
		}
	}
	return nil
}
