package common

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fixtureFile represents the top-level JSON structure.
type fixtureFile struct {
	Fixtures []errorFixture `json:"fixtures"`
}

// errorFixture represents a single test case from testdata/error_fixtures.json.
type errorFixture struct {
	Name         string `json:"name"`
	ChainFamily  string `json:"chain_family"`
	Transport    string `json:"transport"`
	ErrorCode    int    `json:"error_code"`
	ErrorMessage string `json:"error_message"`
	ExpectedCode uint32 `json:"expected_lava_code"`
	ExpectedName string `json:"expected_name"`
}

var chainFamilyFromString = map[string]ChainFamily{
	"EVM":       ChainFamilyEVM,
	"Solana":    ChainFamilySolana,
	"Bitcoin":   ChainFamilyBitcoin,
	"CosmosSDK": ChainFamilyCosmosSDK,
	"Starknet":  ChainFamilyStarknet,
	"Aptos":     ChainFamilyAptos,
	"NEAR":      ChainFamilyNEAR,
	"XRP":       ChainFamilyXRP,
	"Stellar":   ChainFamilyStellar,
	"TON":       ChainFamilyTON,
	"Tron":      ChainFamilyTron,
	"Cardano":   ChainFamilyCardano,
}

var transportFromString = map[string]TransportType{
	"jsonrpc": TransportJsonRPC,
	"rest":    TransportREST,
	"grpc":    TransportGRPC,
}

// TestErrorFixtures loads real error responses from testdata/error_fixtures.json
// and verifies that ClassifyError produces the expected LavaError code for each.
//
// To add a new fixture: add an entry to testdata/error_fixtures.json with the
// actual error code/message from the node and the expected Lava error code.
// This is the primary mechanism for catching classification regressions when
// new node client variants surface in production as UNKNOWN_ERROR.
func TestErrorFixtures(t *testing.T) {
	data, err := os.ReadFile("testdata/error_fixtures.json")
	require.NoError(t, err, "failed to read fixtures file")

	var file fixtureFile
	require.NoError(t, json.Unmarshal(data, &file), "failed to parse fixtures file")
	require.NotEmpty(t, file.Fixtures, "no fixtures found")

	for _, fixture := range file.Fixtures {
		t.Run(fixture.Name, func(t *testing.T) {
			family, ok := chainFamilyFromString[fixture.ChainFamily]
			require.True(t, ok, "unknown chain family in fixture: %s", fixture.ChainFamily)

			transport, ok := transportFromString[fixture.Transport]
			require.True(t, ok, "unknown transport in fixture: %s", fixture.Transport)

			result := ClassifyError(nil, family, transport, fixture.ErrorCode, fixture.ErrorMessage)

			assert.Equal(t, fixture.ExpectedCode, result.Code,
				"fixture %q: expected code %d (%s) but got %d (%s)",
				fixture.Name, fixture.ExpectedCode, fixture.ExpectedName, result.Code, result.Name)
			assert.Equal(t, fixture.ExpectedName, result.Name,
				"fixture %q: expected name %s but got %s",
				fixture.Name, fixture.ExpectedName, result.Name)
		})
	}
}

// TestFixtureFileValid validates the fixture file structure itself.
func TestFixtureFileValid(t *testing.T) {
	data, err := os.ReadFile("testdata/error_fixtures.json")
	require.NoError(t, err)

	var file fixtureFile
	require.NoError(t, json.Unmarshal(data, &file))

	names := make(map[string]bool)
	for _, f := range file.Fixtures {
		// No duplicate names
		assert.False(t, names[f.Name], "duplicate fixture name: %s", f.Name)
		names[f.Name] = true

		// All required fields present
		assert.NotEmpty(t, f.Name, "fixture missing name")
		assert.NotEmpty(t, f.ChainFamily, "fixture %s missing chain_family", f.Name)
		assert.NotEmpty(t, f.Transport, "fixture %s missing transport", f.Name)
		assert.NotEmpty(t, f.ExpectedName, "fixture %s missing expected_name", f.Name)

		// Chain family and transport are valid
		_, ok := chainFamilyFromString[f.ChainFamily]
		assert.True(t, ok, "fixture %s has unknown chain_family: %s", f.Name, f.ChainFamily)
		_, ok = transportFromString[f.Transport]
		assert.True(t, ok, "fixture %s has unknown transport: %s", f.Name, f.Transport)

		// Expected code matches a registered error
		le := GetLavaError(f.ExpectedCode)
		assert.Equal(t, f.ExpectedName, le.Name,
			"fixture %s: expected_lava_code %d resolves to %s, not %s",
			f.Name, f.ExpectedCode, le.Name, f.ExpectedName)
	}
}
