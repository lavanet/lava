package types

import (
	fmt "fmt"

	"cosmossdk.io/collections"
	"cosmossdk.io/collections/indexes"
	"github.com/lavanet/lava/v5/utils"
)

var MetadataVaultIndexesPrefix = collections.NewPrefix([]byte("MetadataVaultIndexes/"))

// ChainIdVaultIndexes defines a secondary unique index for the keeper's stakeEntriesCurrent indexed map
// Normally, a current stake entry can be accessed with the primary key: [chainID, address]
// The new set of indexes, ChainIdVaultIndexes, allows accessing the stake entries with [chainID, vault]
type MetadataVaultIndexes struct {
	Index *indexes.Unique[string, string, ProviderMetadata]
}

func (c MetadataVaultIndexes) IndexesList() []collections.Index[string, ProviderMetadata] {
	return []collections.Index[string, ProviderMetadata]{c.Index}
}

func NewMetadataVaultIndexes(sb *collections.SchemaBuilder) MetadataVaultIndexes {
	return MetadataVaultIndexes{
		Index: indexes.NewUnique(sb, MetadataVaultIndexesPrefix, "provider_metadata_by_vault",
			collections.StringKey,
			collections.StringKey,
			func(pk string, entry ProviderMetadata) (string, error) {
				if entry.Vault == "" {
					return "",
						utils.LavaFormatError("NewMetadataVaultIndexes: cannot create new MetadataVault index",
							fmt.Errorf("empty vault address"),
							utils.LogAttr("provider", entry.Provider),
						)
				}
				return entry.Vault, nil
			},
		),
	}
}
