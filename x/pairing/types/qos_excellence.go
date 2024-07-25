package types

import (
	"cosmossdk.io/collections"
	"cosmossdk.io/collections/indexes"
)

var (
	ProviderClusterQosPrefix  = collections.NewPrefix([]byte("ProviderClusterQos/"))
	ChainClusterQosPrefix     = collections.NewPrefix([]byte("ChainClusterQos/"))
	ChainClusterQosKeysPrefix = collections.NewPrefix([]byte("ChainClusterQosKeys/"))
)

type ChainClusterQosIndexes struct {
	Index *indexes.Multi[collections.Pair[string, string], collections.Triple[string, string, string], ProviderClusterQos]
}

func (c ChainClusterQosIndexes) IndexesList() []collections.Index[collections.Triple[string, string, string], ProviderClusterQos] {
	return []collections.Index[collections.Triple[string, string, string], ProviderClusterQos]{c.Index}
}

func NewChainClusterQosIndexes(sb *collections.SchemaBuilder) ChainClusterQosIndexes {
	return ChainClusterQosIndexes{
		Index: indexes.NewMulti(sb, ChainClusterQosPrefix, "qos_by_chain_cluster",
			collections.PairKeyCodec(collections.StringKey, collections.StringKey),
			collections.TripleKeyCodec(collections.StringKey, collections.StringKey, collections.StringKey),
			func(pk collections.Triple[string, string, string], _ ProviderClusterQos) (collections.Pair[string, string], error) {
				return collections.Join(pk.K1(), pk.K2()), nil
			},
		),
	}
}
