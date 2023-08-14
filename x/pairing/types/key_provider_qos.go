package types

import "strings"

const ProviderQosStorePrefix = "ProviderQosStore/"

func ProviderQosKey(provider string, chainID string, cluster string) string {
	return strings.Join([]string{chainID, provider, cluster}, "/")
}
