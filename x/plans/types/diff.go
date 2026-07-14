package types

import (
	"fmt"
	"strings"
)

// DiffPlans compares two versions of a plan (or two plans in general) and
// returns a human-readable description of every changed field, using the
// plan's JSON field names. The Block field is deliberately ignored: it holds
// the version's creation height, so it differs between any two versions
// without carrying user-visible content.
func DiffPlans(prev, next Plan) []string {
	var changes []string

	appendChange := func(field string, prevVal, nextVal interface{}) {
		changes = append(changes, fmt.Sprintf("%s: %v -> %v", field, prevVal, nextVal))
	}

	if prev.Index != next.Index {
		appendChange("index", quote(prev.Index), quote(next.Index))
	}
	if prev.Description != next.Description {
		appendChange("description", quote(prev.Description), quote(next.Description))
	}
	if prev.Type != next.Type {
		appendChange("type", quote(prev.Type), quote(next.Type))
	}
	if prev.Price.String() != next.Price.String() {
		appendChange("price", prev.Price.String(), next.Price.String())
	}
	if prev.AllowOveruse != next.AllowOveruse {
		appendChange("allow_overuse", prev.AllowOveruse, next.AllowOveruse)
	}
	if prev.OveruseRate != next.OveruseRate {
		appendChange("overuse_rate", prev.OveruseRate, next.OveruseRate)
	}
	if prev.AnnualDiscountPercentage != next.AnnualDiscountPercentage {
		appendChange("annual_discount_percentage", prev.AnnualDiscountPercentage, next.AnnualDiscountPercentage)
	}
	if prev.ProjectsLimit != next.ProjectsLimit {
		appendChange("projects_limit", prev.ProjectsLimit, next.ProjectsLimit)
	}
	if change, changed := diffStringSet("allowed_buyers", prev.AllowedBuyers, next.AllowedBuyers); changed {
		changes = append(changes, change)
	}

	changes = append(changes, diffPolicies(prev.PlanPolicy, next.PlanPolicy)...)

	return changes
}

func diffPolicies(prev, next Policy) []string {
	var changes []string

	appendChange := func(field string, prevVal, nextVal interface{}) {
		changes = append(changes, fmt.Sprintf("plan_policy.%s: %v -> %v", field, prevVal, nextVal))
	}

	changes = append(changes, diffChainPolicies(prev.ChainPolicies, next.ChainPolicies)...)

	if prev.GeolocationProfile != next.GeolocationProfile {
		appendChange("geolocation_profile", geolocationName(prev.GeolocationProfile), geolocationName(next.GeolocationProfile))
	}
	if prev.TotalCuLimit != next.TotalCuLimit {
		appendChange("total_cu_limit", prev.TotalCuLimit, next.TotalCuLimit)
	}
	if prev.EpochCuLimit != next.EpochCuLimit {
		appendChange("epoch_cu_limit", prev.EpochCuLimit, next.EpochCuLimit)
	}
	if prev.MaxProvidersToPair != next.MaxProvidersToPair {
		appendChange("max_providers_to_pair", prev.MaxProvidersToPair, next.MaxProvidersToPair)
	}
	if prev.SelectedProvidersMode != next.SelectedProvidersMode {
		appendChange("selected_providers_mode", prev.SelectedProvidersMode, next.SelectedProvidersMode)
	}
	if change, changed := diffStringSet("plan_policy.selected_providers", prev.SelectedProviders, next.SelectedProviders); changed {
		changes = append(changes, change)
	}

	return changes
}

// diffChainPolicies matches chain policies of both versions by chain id and
// reports added, removed and changed chains (in the order they appear)
func diffChainPolicies(prev, next []ChainPolicy) []string {
	var changes []string

	prevByChain := map[string]ChainPolicy{}
	for _, chainPolicy := range prev {
		prevByChain[chainPolicy.ChainId] = chainPolicy
	}
	nextByChain := map[string]ChainPolicy{}
	for _, chainPolicy := range next {
		nextByChain[chainPolicy.ChainId] = chainPolicy
	}

	for _, nextPolicy := range next {
		prevPolicy, found := prevByChain[nextPolicy.ChainId]
		if !found {
			changes = append(changes, fmt.Sprintf("plan_policy.chain_policies: added %s", FormatChainPolicy(nextPolicy)))
		} else if !prevPolicy.Equal(nextPolicy) {
			changes = append(changes, fmt.Sprintf("plan_policy.chain_policies: changed %s -> %s",
				FormatChainPolicy(prevPolicy), FormatChainPolicy(nextPolicy)))
		}
	}

	for _, prevPolicy := range prev {
		if _, found := nextByChain[prevPolicy.ChainId]; !found {
			changes = append(changes, fmt.Sprintf("plan_policy.chain_policies: removed %s", FormatChainPolicy(prevPolicy)))
		}
	}

	return changes
}

// FormatChainPolicy renders a chain policy in a single compact line, e.g.
// NEAR{requirements: [jsonrpc/POST ext=[archive] mixed]}
func FormatChainPolicy(chainPolicy ChainPolicy) string {
	var parts []string
	if len(chainPolicy.Apis) != 0 {
		parts = append(parts, fmt.Sprintf("apis: [%s]", strings.Join(chainPolicy.Apis, " ")))
	}
	if len(chainPolicy.Requirements) != 0 {
		var requirements []string
		for _, requirement := range chainPolicy.Requirements {
			requirements = append(requirements, formatChainRequirement(requirement))
		}
		parts = append(parts, fmt.Sprintf("requirements: [%s]", strings.Join(requirements, ", ")))
	}
	if len(parts) == 0 {
		return chainPolicy.ChainId + "{}"
	}
	return chainPolicy.ChainId + "{" + strings.Join(parts, ", ") + "}"
}

func formatChainRequirement(requirement ChainRequirement) string {
	collection := requirement.Collection
	str := collection.ApiInterface
	if collection.Type != "" {
		str += "/" + collection.Type
	}
	if collection.InternalPath != "" {
		str += " path=" + collection.InternalPath
	}
	if collection.AddOn != "" {
		str += " addon=" + collection.AddOn
	}
	if len(requirement.Extensions) != 0 {
		str += " ext=[" + strings.Join(requirement.Extensions, " ") + "]"
	}
	if requirement.Mixed {
		str += " mixed"
	}
	return str
}

// diffStringSet reports the strings added to and removed from a field that
// holds a list of unique strings (e.g. allowed_buyers, selected_providers)
func diffStringSet(field string, prev, next []string) (change string, changed bool) {
	prevSet := map[string]struct{}{}
	for _, val := range prev {
		prevSet[val] = struct{}{}
	}
	nextSet := map[string]struct{}{}
	for _, val := range next {
		nextSet[val] = struct{}{}
	}

	var added, removed []string
	for _, val := range next {
		if _, found := prevSet[val]; !found {
			added = append(added, val)
		}
	}
	for _, val := range prev {
		if _, found := nextSet[val]; !found {
			removed = append(removed, val)
		}
	}

	if len(added) == 0 && len(removed) == 0 {
		return "", false
	}

	var parts []string
	if len(added) != 0 {
		parts = append(parts, fmt.Sprintf("added [%s]", strings.Join(added, " ")))
	}
	if len(removed) != 0 {
		parts = append(parts, fmt.Sprintf("removed [%s]", strings.Join(removed, " ")))
	}

	return fmt.Sprintf("%s: %s", field, strings.Join(parts, ", ")), true
}

// geolocationName renders a geolocation bitmask using the enum names (falls
// back to the raw value if a bit does not match any known region)
func geolocationName(geoloc int32) string {
	if name, ok := Geolocation_name[geoloc]; ok {
		return name
	}
	if !IsValidGeoEnum(geoloc) {
		return fmt.Sprintf("%d", geoloc)
	}

	var names []string
	for _, geo := range GetGeolocationsFromUint(geoloc) {
		names = append(names, Geolocation_name[int32(geo)])
	}
	return strings.Join(names, ",")
}

func quote(s string) string {
	return `"` + s + `"`
}
