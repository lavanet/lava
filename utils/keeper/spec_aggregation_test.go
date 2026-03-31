package keeper

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	types "github.com/lavanet/lava/v5/types/spec"
	"github.com/stretchr/testify/require"
)

func TestGetAllSpecsFromFile(t *testing.T) {
	// Create a temp directory for test files
	tmpDir := t.TempDir()

	// Create a test spec file
	specFile := filepath.Join(tmpDir, "test_spec.json")
	specContent := `{
		"proposal": {
			"specs": [
				{
					"index": "TEST1",
					"name": "Test Chain 1",
					"enabled": true
				},
				{
					"index": "TEST2",
					"name": "Test Chain 2",
					"enabled": true
				}
			]
		}
	}`
	err := os.WriteFile(specFile, []byte(specContent), 0o644)
	require.NoError(t, err)

	// Test loading specs from file
	specs, err := GetAllSpecsFromFile(specFile)
	require.NoError(t, err)
	require.Len(t, specs, 2)
	require.Contains(t, specs, "TEST1")
	require.Contains(t, specs, "TEST2")
	require.Equal(t, "Test Chain 1", specs["TEST1"].Name)
	require.Equal(t, "Test Chain 2", specs["TEST2"].Name)
}

func TestGetAllSpecsFromLocalDir(t *testing.T) {
	// Create a temp directory for test files
	tmpDir := t.TempDir()

	// Create first spec file
	specFile1 := filepath.Join(tmpDir, "spec1.json")
	specContent1 := `{
		"proposal": {
			"specs": [
				{
					"index": "CHAIN1",
					"name": "Chain One",
					"enabled": true
				}
			]
		}
	}`
	err := os.WriteFile(specFile1, []byte(specContent1), 0o644)
	require.NoError(t, err)

	// Create second spec file
	specFile2 := filepath.Join(tmpDir, "spec2.json")
	specContent2 := `{
		"proposal": {
			"specs": [
				{
					"index": "CHAIN2",
					"name": "Chain Two",
					"enabled": true
				}
			]
		}
	}`
	err = os.WriteFile(specFile2, []byte(specContent2), 0o644)
	require.NoError(t, err)

	// Test loading all specs from directory
	specs, err := GetAllSpecsFromLocalDir(tmpDir)
	require.NoError(t, err)
	require.Len(t, specs, 2)
	require.Contains(t, specs, "CHAIN1")
	require.Contains(t, specs, "CHAIN2")
}

func TestGetAllSpecsFromPath(t *testing.T) {
	// Create a temp directory for test files
	tmpDir := t.TempDir()

	// Create a test spec file
	specFile := filepath.Join(tmpDir, "test_spec.json")
	specContent := `{
		"proposal": {
			"specs": [
				{
					"index": "PATH_TEST",
					"name": "Path Test Chain",
					"enabled": true
				}
			]
		}
	}`
	err := os.WriteFile(specFile, []byte(specContent), 0o644)
	require.NoError(t, err)

	t.Run("from file", func(t *testing.T) {
		specs, err := GetAllSpecsFromPath(specFile)
		require.NoError(t, err)
		require.Len(t, specs, 1)
		require.Contains(t, specs, "PATH_TEST")
	})

	t.Run("from directory", func(t *testing.T) {
		specs, err := GetAllSpecsFromPath(tmpDir)
		require.NoError(t, err)
		require.Len(t, specs, 1)
		require.Contains(t, specs, "PATH_TEST")
	})

	t.Run("non-existent path", func(t *testing.T) {
		_, err := GetAllSpecsFromPath("/non/existent/path")
		require.Error(t, err)
	})
}

func TestExpandSpecWithDependencies(t *testing.T) {
	// Create specs map with base and derived specs
	specs := map[string]types.Spec{
		"BASE": {
			Index:   "BASE",
			Name:    "Base Chain",
			Enabled: true,
		},
		"DERIVED": {
			Index:   "DERIVED",
			Name:    "Derived Chain",
			Enabled: true,
			// Note: In a real scenario, this would have inheritance
		},
	}

	t.Run("expand existing spec", func(t *testing.T) {
		spec, err := ExpandSpecWithDependencies(specs, "BASE")
		require.NoError(t, err)
		require.Equal(t, "BASE", spec.Index)
		require.Equal(t, "Base Chain", spec.Name)
	})

	t.Run("expand non-existent spec", func(t *testing.T) {
		_, err := ExpandSpecWithDependencies(specs, "NONEXISTENT")
		require.Error(t, err)
		require.Contains(t, err.Error(), "spec not found")
	})
}

func TestSpecAggregation_OverrideBehavior(t *testing.T) {
	// Create a temp directory for test files
	tmpDir := t.TempDir()

	// Create first source directory
	source1 := filepath.Join(tmpDir, "source1")
	err := os.MkdirAll(source1, 0o755)
	require.NoError(t, err)

	// Create second source directory
	source2 := filepath.Join(tmpDir, "source2")
	err = os.MkdirAll(source2, 0o755)
	require.NoError(t, err)

	// Write spec to source1 with initial values
	spec1Content := `{
		"proposal": {
			"specs": [
				{
					"index": "SHARED",
					"name": "Original Name",
					"enabled": true
				},
				{
					"index": "SOURCE1_ONLY",
					"name": "Source 1 Only",
					"enabled": true
				}
			]
		}
	}`
	err = os.WriteFile(filepath.Join(source1, "spec.json"), []byte(spec1Content), 0o644)
	require.NoError(t, err)

	// Write spec to source2 with overriding values
	spec2Content := `{
		"proposal": {
			"specs": [
				{
					"index": "SHARED",
					"name": "Overridden Name",
					"enabled": false
				},
				{
					"index": "SOURCE2_ONLY",
					"name": "Source 2 Only",
					"enabled": true
				}
			]
		}
	}`
	err = os.WriteFile(filepath.Join(source2, "spec.json"), []byte(spec2Content), 0o644)
	require.NoError(t, err)

	// Load specs from both sources
	specs1, err := GetAllSpecsFromPath(source1)
	require.NoError(t, err)

	specs2, err := GetAllSpecsFromPath(source2)
	require.NoError(t, err)

	// Aggregate: source2 should override source1
	aggregated := make(map[string]types.Spec)
	for k, v := range specs1 {
		aggregated[k] = v
	}
	for k, v := range specs2 {
		aggregated[k] = v // This simulates the override behavior
	}

	// Verify aggregation
	require.Len(t, aggregated, 3) // SHARED, SOURCE1_ONLY, SOURCE2_ONLY

	// Verify SHARED was overridden by source2
	require.Equal(t, "Overridden Name", aggregated["SHARED"].Name)
	require.False(t, aggregated["SHARED"].Enabled)

	// Verify SOURCE1_ONLY is still present
	require.Contains(t, aggregated, "SOURCE1_ONLY")
	require.Equal(t, "Source 1 Only", aggregated["SOURCE1_ONLY"].Name)

	// Verify SOURCE2_ONLY is present
	require.Contains(t, aggregated, "SOURCE2_ONLY")
	require.Equal(t, "Source 2 Only", aggregated["SOURCE2_ONLY"].Name)
}

// TestSpecAggregation_FullReplacement verifies that when specs have the same index,
// the later spec COMPLETELY replaces the earlier one - including api_collections,
// min_stake_provider, and all other fields. This is NOT a merge operation.
func TestSpecAggregation_FullReplacement(t *testing.T) {
	// Create a temp directory for test files
	tmpDir := t.TempDir()

	// Create first source directory
	source1 := filepath.Join(tmpDir, "source1")
	err := os.MkdirAll(source1, 0o755)
	require.NoError(t, err)

	// Create second source directory
	source2 := filepath.Join(tmpDir, "source2")
	err = os.MkdirAll(source2, 0o755)
	require.NoError(t, err)

	// Source 1: Full spec with many APIs, high min_stake, specific settings
	spec1Content := `{
		"proposal": {
			"specs": [
				{
					"index": "ETH1",
					"name": "Ethereum Mainnet Original",
					"enabled": true,
					"reliability_threshold": 268435455,
					"data_reliability_enabled": true,
					"block_distance_for_finalized_data": 8,
					"blocks_in_finalization_proof": 3,
					"average_block_time": 13000,
					"allowed_block_lag_for_qos_sync": 2,
					"shares": 1,
					"min_stake_provider": {
						"denom": "ulava",
						"amount": "5000000000"
					},
					"api_collections": [
						{
							"enabled": true,
							"collection_data": {
								"api_interface": "jsonrpc",
								"internal_path": "",
								"type": "POST",
								"add_on": ""
							},
							"apis": [
								{
									"name": "eth_blockNumber",
									"compute_units": 10,
									"enabled": true
								},
								{
									"name": "eth_getBalance",
									"compute_units": 20,
									"enabled": true
								},
								{
									"name": "eth_call",
									"compute_units": 30,
									"enabled": true
								}
							]
						}
					]
				}
			]
		}
	}`
	err = os.WriteFile(filepath.Join(source1, "ethereum.json"), []byte(spec1Content), 0o644)
	require.NoError(t, err)

	// Source 2: Completely different spec for same chain - fewer APIs, different settings
	// This should REPLACE source1's spec entirely, NOT merge with it
	spec2Content := `{
		"proposal": {
			"specs": [
				{
					"index": "ETH1",
					"name": "Ethereum Mainnet Override",
					"enabled": false,
					"reliability_threshold": 100,
					"data_reliability_enabled": false,
					"block_distance_for_finalized_data": 1,
					"average_block_time": 12000,
					"shares": 5,
					"min_stake_provider": {
						"denom": "ulava",
						"amount": "1000000"
					},
					"api_collections": [
						{
							"enabled": true,
							"collection_data": {
								"api_interface": "jsonrpc",
								"internal_path": "",
								"type": "POST",
								"add_on": ""
							},
							"apis": [
								{
									"name": "eth_chainId",
									"compute_units": 5,
									"enabled": true
								}
							]
						}
					]
				}
			]
		}
	}`
	err = os.WriteFile(filepath.Join(source2, "ethereum_override.json"), []byte(spec2Content), 0o644)
	require.NoError(t, err)

	// Load specs from both sources
	specs1, err := GetAllSpecsFromPath(source1)
	require.NoError(t, err)

	specs2, err := GetAllSpecsFromPath(source2)
	require.NoError(t, err)

	// Verify source1 has 3 APIs
	require.Len(t, specs1["ETH1"].ApiCollections, 1)
	require.Len(t, specs1["ETH1"].ApiCollections[0].Apis, 3)
	require.Equal(t, "eth_blockNumber", specs1["ETH1"].ApiCollections[0].Apis[0].Name)
	require.Equal(t, "Ethereum Mainnet Original", specs1["ETH1"].Name)
	require.True(t, specs1["ETH1"].Enabled)
	require.Equal(t, int64(5000000000), specs1["ETH1"].MinStakeProvider.Amount)

	// Verify source2 has only 1 API
	require.Len(t, specs2["ETH1"].ApiCollections, 1)
	require.Len(t, specs2["ETH1"].ApiCollections[0].Apis, 1)
	require.Equal(t, "eth_chainId", specs2["ETH1"].ApiCollections[0].Apis[0].Name)

	// Aggregate: source2 should COMPLETELY replace source1
	aggregated := make(map[string]types.Spec)
	for k, v := range specs1 {
		aggregated[k] = v
	}
	for k, v := range specs2 {
		aggregated[k] = v // Full replacement, not merge
	}

	// Verify the aggregated spec is COMPLETELY from source2
	finalSpec := aggregated["ETH1"]

	// Name should be from source2
	require.Equal(t, "Ethereum Mainnet Override", finalSpec.Name)

	// Enabled should be from source2 (false)
	require.False(t, finalSpec.Enabled)

	// min_stake_provider should be from source2 (lower amount)
	require.Equal(t, int64(1000000), finalSpec.MinStakeProvider.Amount)

	// reliability_threshold should be from source2
	require.Equal(t, uint32(100), finalSpec.ReliabilityThreshold)

	// shares should be from source2
	require.Equal(t, uint64(5), finalSpec.Shares)

	// average_block_time should be from source2
	require.Equal(t, int64(12000), finalSpec.AverageBlockTime)

	// CRITICAL: api_collections should be COMPLETELY from source2
	// Source1 had 3 APIs, source2 has only 1 - we should have ONLY 1 API
	require.Len(t, finalSpec.ApiCollections, 1, "Should have exactly 1 api_collection from source2")
	require.Len(t, finalSpec.ApiCollections[0].Apis, 1, "Should have exactly 1 API from source2, NOT merged with source1's 3 APIs")
	require.Equal(t, "eth_chainId", finalSpec.ApiCollections[0].Apis[0].Name, "The only API should be eth_chainId from source2")

	// Verify source1's APIs are NOT present (proves full replacement, not merge)
	for _, api := range finalSpec.ApiCollections[0].Apis {
		require.NotEqual(t, "eth_blockNumber", api.Name, "eth_blockNumber from source1 should NOT be present")
		require.NotEqual(t, "eth_getBalance", api.Name, "eth_getBalance from source1 should NOT be present")
		require.NotEqual(t, "eth_call", api.Name, "eth_call from source1 should NOT be present")
	}
}

func TestGetAllSpecsFromLocalDir_IgnoresNonJSON(t *testing.T) {
	// Create a temp directory for test files
	tmpDir := t.TempDir()

	// Create a JSON spec file
	specFile := filepath.Join(tmpDir, "spec.json")
	specContent := `{
		"proposal": {
			"specs": [
				{
					"index": "VALID",
					"name": "Valid Spec",
					"enabled": true
				}
			]
		}
	}`
	err := os.WriteFile(specFile, []byte(specContent), 0o644)
	require.NoError(t, err)

	// Create a non-JSON file (should be ignored)
	readmeFile := filepath.Join(tmpDir, "README.md")
	err = os.WriteFile(readmeFile, []byte("# Readme"), 0o644)
	require.NoError(t, err)

	// Create a txt file (should be ignored)
	txtFile := filepath.Join(tmpDir, "notes.txt")
	err = os.WriteFile(txtFile, []byte("Some notes"), 0o644)
	require.NoError(t, err)

	// Load specs - should only load from JSON file
	specs, err := GetAllSpecsFromLocalDir(tmpDir)
	require.NoError(t, err)
	require.Len(t, specs, 1)
	require.Contains(t, specs, "VALID")
}

// TestSpecAggregation_MultipleLocalFiles tests aggregation from multiple local files
// (simulating --use-static-spec file1.json --use-static-spec file2.json)
// Note: Comma-separated values are NOT supported. Each source must be a separate flag.
func TestSpecAggregation_MultipleLocalFiles(t *testing.T) {
	tmpDir := t.TempDir()

	// Create first spec file - base specs
	file1 := filepath.Join(tmpDir, "base_specs.json")
	file1Content := `{
		"proposal": {
			"specs": [
				{
					"index": "ETH1",
					"name": "Ethereum from file1",
					"enabled": true,
					"shares": 10,
					"api_collections": [
						{
							"enabled": true,
							"collection_data": {"api_interface": "jsonrpc", "type": "POST"},
							"apis": [{"name": "eth_blockNumber", "compute_units": 10, "enabled": true}]
						}
					]
				},
				{
					"index": "COSMOS",
					"name": "Cosmos Hub from file1",
					"enabled": true,
					"shares": 5
				}
			]
		}
	}`
	err := os.WriteFile(file1, []byte(file1Content), 0o644)
	require.NoError(t, err)

	// Create second spec file - overrides ETH1
	file2 := filepath.Join(tmpDir, "override_eth.json")
	file2Content := `{
		"proposal": {
			"specs": [
				{
					"index": "ETH1",
					"name": "Ethereum from file2 (override)",
					"enabled": false,
					"shares": 100,
					"api_collections": [
						{
							"enabled": true,
							"collection_data": {"api_interface": "jsonrpc", "type": "POST"},
							"apis": [
								{"name": "eth_chainId", "compute_units": 5, "enabled": true},
								{"name": "eth_gasPrice", "compute_units": 15, "enabled": true}
							]
						}
					]
				}
			]
		}
	}`
	err = os.WriteFile(file2, []byte(file2Content), 0o644)
	require.NoError(t, err)

	// Create third spec file - adds new chain
	file3 := filepath.Join(tmpDir, "extra_chain.json")
	file3Content := `{
		"proposal": {
			"specs": [
				{
					"index": "SOL",
					"name": "Solana from file3",
					"enabled": true,
					"shares": 20
				}
			]
		}
	}`
	err = os.WriteFile(file3, []byte(file3Content), 0o644)
	require.NoError(t, err)

	// Simulate: --use-static-spec file1.json --use-static-spec file2.json --use-static-spec file3.json
	sources := []string{file1, file2, file3}

	// Aggregate from all sources in order
	aggregated := make(map[string]types.Spec)
	for _, source := range sources {
		specs, err := GetAllSpecsFromPath(source)
		require.NoError(t, err)
		for chainID, spec := range specs {
			aggregated[chainID] = spec // Last wins
		}
	}

	// Verify we have 3 chains: ETH1, COSMOS, SOL
	require.Len(t, aggregated, 3)
	require.Contains(t, aggregated, "ETH1")
	require.Contains(t, aggregated, "COSMOS")
	require.Contains(t, aggregated, "SOL")

	// Verify ETH1 is completely from file2 (not merged with file1)
	eth := aggregated["ETH1"]
	require.Equal(t, "Ethereum from file2 (override)", eth.Name)
	require.False(t, eth.Enabled)
	require.Equal(t, uint64(100), eth.Shares)
	require.Len(t, eth.ApiCollections, 1)
	require.Len(t, eth.ApiCollections[0].Apis, 2) // Only file2's 2 APIs, not merged with file1's 1 API
	require.Equal(t, "eth_chainId", eth.ApiCollections[0].Apis[0].Name)
	require.Equal(t, "eth_gasPrice", eth.ApiCollections[0].Apis[1].Name)

	// Verify COSMOS remains from file1 (untouched)
	cosmos := aggregated["COSMOS"]
	require.Equal(t, "Cosmos Hub from file1", cosmos.Name)
	require.True(t, cosmos.Enabled)
	require.Equal(t, uint64(5), cosmos.Shares)

	// Verify SOL is from file3
	sol := aggregated["SOL"]
	require.Equal(t, "Solana from file3", sol.Name)
	require.True(t, sol.Enabled)
	require.Equal(t, uint64(20), sol.Shares)
}

// TestSpecAggregation_RemoteAndLocalFile tests aggregation between a simulated remote source
// (GitHub/GitLab) and a local file. Remote is loaded first, then local file overrides.
// This simulates: --use-static-spec https://github.com/... --use-static-spec ./local.json
func TestSpecAggregation_RemoteAndLocalFile(t *testing.T) {
	// Create local override file
	tmpDir := t.TempDir()
	localFile := filepath.Join(tmpDir, "local_override.json")
	localContent := `{
		"proposal": {
			"specs": [
				{
					"index": "ETH1",
					"name": "Ethereum from Local Override",
					"enabled": false,
					"shares": 999,
					"api_collections": [
						{
							"enabled": true,
							"collection_data": {"api_interface": "jsonrpc", "type": "POST"},
							"apis": [{"name": "eth_localOnly", "compute_units": 1, "enabled": true}]
						}
					]
				}
			]
		}
	}`
	err := os.WriteFile(localFile, []byte(localContent), 0o644)
	require.NoError(t, err)

	// Simulate what would come from a remote GitHub/GitLab repository
	// In production, this would be fetched via specfetcher.FetchAllSpecsFromRemote()
	remoteSpecs := map[string]types.Spec{
		"ETH1": {
			Index:   "ETH1",
			Name:    "Ethereum from Remote GitHub",
			Enabled: true,
			Shares:  50,
			ApiCollections: []*types.ApiCollection{
				{
					Enabled:        true,
					CollectionData: types.CollectionData{ApiInterface: "jsonrpc", Type: "POST"},
					Apis: []*types.Api{
						{Name: "eth_blockNumber", ComputeUnits: 10, Enabled: true},
						{Name: "eth_getBlock", ComputeUnits: 20, Enabled: true},
					},
				},
			},
		},
		"AVAX": {
			Index:   "AVAX",
			Name:    "Avalanche from Remote",
			Enabled: true,
			Shares:  30,
		},
	}

	// Load from local file
	localSpecs, err := GetAllSpecsFromPath(localFile)
	require.NoError(t, err)

	// Aggregate: remote first, then local override
	// This simulates: --use-static-spec https://github.com/... --use-static-spec ./local.json
	aggregated := make(map[string]types.Spec)
	for k, v := range remoteSpecs {
		aggregated[k] = v
	}
	for k, v := range localSpecs {
		aggregated[k] = v // Local wins over remote
	}

	// Verify: ETH1 should be completely from local, AVAX from remote
	require.Len(t, aggregated, 2)

	// ETH1 should be the local version (fully replaced, not merged)
	eth := aggregated["ETH1"]
	require.Equal(t, "Ethereum from Local Override", eth.Name)
	require.False(t, eth.Enabled)
	require.Equal(t, uint64(999), eth.Shares)
	require.Len(t, eth.ApiCollections, 1)
	require.Len(t, eth.ApiCollections[0].Apis, 1)
	require.Equal(t, "eth_localOnly", eth.ApiCollections[0].Apis[0].Name)

	// AVAX should remain from remote (not overridden)
	avax := aggregated["AVAX"]
	require.Equal(t, "Avalanche from Remote", avax.Name)
	require.True(t, avax.Enabled)
	require.Equal(t, uint64(30), avax.Shares)
}

// TestSpecAggregation_RemoteAndLocalDirectory tests aggregation between a mocked remote source
// and a local directory containing multiple spec files.
func TestSpecAggregation_RemoteAndLocalDirectory(t *testing.T) {
	// Simulate remote specs (what would come from GitHub/GitLab)
	remoteSpecs := map[string]types.Spec{
		"ETH1": {
			Index:   "ETH1",
			Name:    "Ethereum from Remote",
			Enabled: true,
			Shares:  50,
			ApiCollections: []*types.ApiCollection{
				{
					Enabled:        true,
					CollectionData: types.CollectionData{ApiInterface: "jsonrpc", Type: "POST"},
					Apis: []*types.Api{
						{Name: "eth_blockNumber", ComputeUnits: 10, Enabled: true},
						{Name: "eth_getBalance", ComputeUnits: 20, Enabled: true},
						{Name: "eth_call", ComputeUnits: 30, Enabled: true},
					},
				},
			},
		},
		"AVAX": {
			Index:   "AVAX",
			Name:    "Avalanche from Remote",
			Enabled: true,
			Shares:  30,
		},
		"SOL": {
			Index:   "SOL",
			Name:    "Solana from Remote",
			Enabled: true,
			Shares:  25,
		},
	}

	// Create local directory with override specs
	tmpDir := t.TempDir()
	localDir := filepath.Join(tmpDir, "local_overrides")
	err := os.MkdirAll(localDir, 0o755)
	require.NoError(t, err)

	// Local file 1: Override ETH1
	ethOverride := filepath.Join(localDir, "eth_override.json")
	ethContent := `{
		"proposal": {
			"specs": [
				{
					"index": "ETH1",
					"name": "Ethereum Local Override",
					"enabled": false,
					"shares": 1000,
					"api_collections": [
						{
							"enabled": true,
							"collection_data": {"api_interface": "jsonrpc", "type": "POST"},
							"apis": [{"name": "eth_custom", "compute_units": 1, "enabled": true}]
						}
					]
				}
			]
		}
	}`
	err = os.WriteFile(ethOverride, []byte(ethContent), 0o644)
	require.NoError(t, err)

	// Local file 2: Override SOL and add new chain
	solOverride := filepath.Join(localDir, "sol_and_new.json")
	solContent := `{
		"proposal": {
			"specs": [
				{
					"index": "SOL",
					"name": "Solana Local Override",
					"enabled": false,
					"shares": 500
				},
				{
					"index": "MATIC",
					"name": "Polygon from Local",
					"enabled": true,
					"shares": 75
				}
			]
		}
	}`
	err = os.WriteFile(solOverride, []byte(solContent), 0o644)
	require.NoError(t, err)

	// Simulate: --use-static-spec https://github.com/org/specs/tree/main/specs
	//           --use-static-spec ./local_overrides/

	// Load from local directory
	localSpecs, err := GetAllSpecsFromPath(localDir)
	require.NoError(t, err)

	// Aggregate: remote first, then local directory overrides
	aggregated := make(map[string]types.Spec)
	for k, v := range remoteSpecs {
		aggregated[k] = v
	}
	for k, v := range localSpecs {
		aggregated[k] = v // Local directory wins over remote
	}

	// Verify we have 4 chains: ETH1, AVAX, SOL, MATIC
	require.Len(t, aggregated, 4)

	// ETH1: fully replaced by local
	eth := aggregated["ETH1"]
	require.Equal(t, "Ethereum Local Override", eth.Name)
	require.False(t, eth.Enabled)
	require.Equal(t, uint64(1000), eth.Shares)
	require.Len(t, eth.ApiCollections, 1)
	require.Len(t, eth.ApiCollections[0].Apis, 1)
	require.Equal(t, "eth_custom", eth.ApiCollections[0].Apis[0].Name)

	// AVAX: unchanged from remote
	avax := aggregated["AVAX"]
	require.Equal(t, "Avalanche from Remote", avax.Name)
	require.True(t, avax.Enabled)
	require.Equal(t, uint64(30), avax.Shares)

	// SOL: fully replaced by local
	sol := aggregated["SOL"]
	require.Equal(t, "Solana Local Override", sol.Name)
	require.False(t, sol.Enabled)
	require.Equal(t, uint64(500), sol.Shares)

	// MATIC: new chain from local only
	matic := aggregated["MATIC"]
	require.Equal(t, "Polygon from Local", matic.Name)
	require.True(t, matic.Enabled)
	require.Equal(t, uint64(75), matic.Shares)
}

// TestSpecAggregation_MultipleRemoteSources tests aggregation between multiple remote sources
// (GitHub + GitLab) followed by local overrides.
func TestSpecAggregation_MultipleRemoteSources(t *testing.T) {
	// Simulate GitHub remote specs
	githubSpecs := map[string]types.Spec{
		"ETH1": {
			Index:   "ETH1",
			Name:    "Ethereum from GitHub",
			Enabled: true,
			Shares:  100,
		},
		"AVAX": {
			Index:   "AVAX",
			Name:    "Avalanche from GitHub",
			Enabled: true,
			Shares:  50,
		},
	}

	// Simulate GitLab remote specs (overrides ETH1, adds COSMOS)
	gitlabSpecs := map[string]types.Spec{
		"ETH1": {
			Index:   "ETH1",
			Name:    "Ethereum from GitLab (override)",
			Enabled: false,
			Shares:  200,
		},
		"COSMOS": {
			Index:   "COSMOS",
			Name:    "Cosmos from GitLab",
			Enabled: true,
			Shares:  75,
		},
	}

	// Create local override file
	tmpDir := t.TempDir()
	localFile := filepath.Join(tmpDir, "local.json")
	localContent := `{
		"proposal": {
			"specs": [
				{
					"index": "ETH1",
					"name": "Ethereum from Local (final override)",
					"enabled": true,
					"shares": 9999
				}
			]
		}
	}`
	err := os.WriteFile(localFile, []byte(localContent), 0o644)
	require.NoError(t, err)

	// Simulate: --use-static-spec https://github.com/...
	//           --use-static-spec https://gitlab.com/...
	//           --use-static-spec ./local.json

	localSpecs, err := GetAllSpecsFromPath(localFile)
	require.NoError(t, err)

	// Aggregate in order: GitHub -> GitLab -> Local
	aggregated := make(map[string]types.Spec)

	// First: GitHub
	for k, v := range githubSpecs {
		aggregated[k] = v
	}

	// Second: GitLab (overrides GitHub)
	for k, v := range gitlabSpecs {
		aggregated[k] = v
	}

	// Third: Local (overrides both)
	for k, v := range localSpecs {
		aggregated[k] = v
	}

	// Verify we have 3 chains: ETH1, AVAX, COSMOS
	require.Len(t, aggregated, 3)

	// ETH1: went through 3 versions, final is local
	eth := aggregated["ETH1"]
	require.Equal(t, "Ethereum from Local (final override)", eth.Name)
	require.True(t, eth.Enabled)
	require.Equal(t, uint64(9999), eth.Shares)

	// AVAX: only from GitHub, not overridden
	avax := aggregated["AVAX"]
	require.Equal(t, "Avalanche from GitHub", avax.Name)
	require.True(t, avax.Enabled)
	require.Equal(t, uint64(50), avax.Shares)

	// COSMOS: only from GitLab, not overridden
	cosmos := aggregated["COSMOS"]
	require.Equal(t, "Cosmos from GitLab", cosmos.Name)
	require.True(t, cosmos.Enabled)
	require.Equal(t, uint64(75), cosmos.Shares)
}

// TestSpecAggregation_OrderMatters verifies that the order of sources determines
// which spec wins when there are conflicts.
func TestSpecAggregation_OrderMatters(t *testing.T) {
	tmpDir := t.TempDir()

	// Create spec A
	specA := filepath.Join(tmpDir, "a.json")
	contentA := `{"proposal":{"specs":[{"index":"TEST","name":"From A","shares":1}]}}`
	require.NoError(t, os.WriteFile(specA, []byte(contentA), 0o644))

	// Create spec B
	specB := filepath.Join(tmpDir, "b.json")
	contentB := `{"proposal":{"specs":[{"index":"TEST","name":"From B","shares":2}]}}`
	require.NoError(t, os.WriteFile(specB, []byte(contentB), 0o644))

	// Create spec C
	specC := filepath.Join(tmpDir, "c.json")
	contentC := `{"proposal":{"specs":[{"index":"TEST","name":"From C","shares":3}]}}`
	require.NoError(t, os.WriteFile(specC, []byte(contentC), 0o644))

	t.Run("order A-B-C results in C winning", func(t *testing.T) {
		aggregated := aggregateSpecsInOrder(t, []string{specA, specB, specC})
		require.Equal(t, "From C", aggregated["TEST"].Name)
		require.Equal(t, uint64(3), aggregated["TEST"].Shares)
	})

	t.Run("order C-B-A results in A winning", func(t *testing.T) {
		aggregated := aggregateSpecsInOrder(t, []string{specC, specB, specA})
		require.Equal(t, "From A", aggregated["TEST"].Name)
		require.Equal(t, uint64(1), aggregated["TEST"].Shares)
	})

	t.Run("order A-C-B results in B winning", func(t *testing.T) {
		aggregated := aggregateSpecsInOrder(t, []string{specA, specC, specB})
		require.Equal(t, "From B", aggregated["TEST"].Name)
		require.Equal(t, uint64(2), aggregated["TEST"].Shares)
	})
}

// aggregateSpecsInOrder is a helper that loads specs from multiple sources in order
// and aggregates them with last-wins semantics.
func aggregateSpecsInOrder(t *testing.T, sources []string) map[string]types.Spec {
	t.Helper()
	aggregated := make(map[string]types.Spec)
	for _, source := range sources {
		specs, err := GetAllSpecsFromPath(source)
		require.NoError(t, err)
		for k, v := range specs {
			aggregated[k] = v
		}
	}
	return aggregated
}

// TestSpecAggregation_CommaSeparatedPaths tests that comma-separated local file paths
// are correctly expanded and processed. This simulates:
//
//	--use-static-spec file1.json,file2.json,file3.json
//
// which should behave the same as:
//
//	--use-static-spec file1.json --use-static-spec file2.json --use-static-spec file3.json
func TestSpecAggregation_CommaSeparatedPaths(t *testing.T) {
	tmpDir := t.TempDir()

	// Create three spec files
	file1 := filepath.Join(tmpDir, "spec1.json")
	content1 := `{"proposal":{"specs":[{"index":"ETH1","name":"From spec1","shares":1}]}}`
	require.NoError(t, os.WriteFile(file1, []byte(content1), 0o644))

	file2 := filepath.Join(tmpDir, "spec2.json")
	content2 := `{"proposal":{"specs":[{"index":"ETH1","name":"From spec2","shares":2},{"index":"AVAX","name":"Avax from spec2","shares":10}]}}`
	require.NoError(t, os.WriteFile(file2, []byte(content2), 0o644))

	file3 := filepath.Join(tmpDir, "spec3.json")
	content3 := `{"proposal":{"specs":[{"index":"ETH1","name":"From spec3","shares":3}]}}`
	require.NoError(t, os.WriteFile(file3, []byte(content3), 0o644))

	// Test expandCommaSeparatedPaths function behavior
	// This simulates what state_tracker.go does with comma-separated values
	t.Run("comma-separated paths expand correctly", func(t *testing.T) {
		// Simulate: --use-static-spec "file1.json,file2.json,file3.json"
		commaSeparated := file1 + "," + file2 + "," + file3
		expanded := expandCommaSeparatedLocalPaths(commaSeparated)

		require.Len(t, expanded, 3)
		require.Equal(t, file1, expanded[0])
		require.Equal(t, file2, expanded[1])
		require.Equal(t, file3, expanded[2])
	})

	t.Run("comma-separated with spaces", func(t *testing.T) {
		// Simulate: --use-static-spec "file1.json, file2.json , file3.json"
		commaSeparated := file1 + ", " + file2 + " , " + file3
		expanded := expandCommaSeparatedLocalPaths(commaSeparated)

		require.Len(t, expanded, 3)
		require.Equal(t, file1, expanded[0])
		require.Equal(t, file2, expanded[1])
		require.Equal(t, file3, expanded[2])
	})

	t.Run("aggregation works with comma-separated paths", func(t *testing.T) {
		// Simulate the aggregation that state_tracker does after expanding
		commaSeparated := file1 + "," + file2 + "," + file3
		expanded := expandCommaSeparatedLocalPaths(commaSeparated)

		aggregated := make(map[string]types.Spec)
		for _, path := range expanded {
			specs, err := GetAllSpecsFromPath(path)
			require.NoError(t, err)
			for k, v := range specs {
				aggregated[k] = v
			}
		}

		// ETH1 should be from file3 (last wins)
		require.Equal(t, "From spec3", aggregated["ETH1"].Name)
		require.Equal(t, uint64(3), aggregated["ETH1"].Shares)

		// AVAX should be from file2 (only source)
		require.Equal(t, "Avax from spec2", aggregated["AVAX"].Name)
		require.Equal(t, uint64(10), aggregated["AVAX"].Shares)
	})

	t.Run("mixed separate flags and comma-separated", func(t *testing.T) {
		// Simulate: --use-static-spec "file1.json,file2.json" --use-static-spec file3.json
		// After StringArray parsing, we get: ["file1.json,file2.json", "file3.json"]
		flagValues := []string{file1 + "," + file2, file3}

		var allPaths []string
		for _, val := range flagValues {
			allPaths = append(allPaths, expandCommaSeparatedLocalPaths(val)...)
		}

		require.Len(t, allPaths, 3)

		aggregated := make(map[string]types.Spec)
		for _, path := range allPaths {
			specs, err := GetAllSpecsFromPath(path)
			require.NoError(t, err)
			for k, v := range specs {
				aggregated[k] = v
			}
		}

		// ETH1 should be from file3 (last wins)
		require.Equal(t, "From spec3", aggregated["ETH1"].Name)
		require.Equal(t, uint64(3), aggregated["ETH1"].Shares)
	})
}

// expandCommaSeparatedLocalPaths is a test helper that mimics the behavior
// of state_tracker.expandCommaSeparatedPaths for local paths only.
func expandCommaSeparatedLocalPaths(path string) []string {
	var expanded []string
	parts := strings.Split(path, ",")
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed != "" {
			expanded = append(expanded, trimmed)
		}
	}
	return expanded
}
