package keeper_test

import (
	"testing"

	"github.com/lavanet/lava/testutil/common"
	epochstoragetypes "github.com/lavanet/lava/x/epochstorage/types"
	"github.com/lavanet/lava/x/pairing/client/cli"
	spectypes "github.com/lavanet/lava/x/spec/types"
	"github.com/stretchr/testify/require"
)

// Test that the optional moniker argument in StakeProvider doesn't break anything
func TestStakeProviderWithMoniker(t *testing.T) {
	ts := newTester(t)

	epochDetails := *epochstoragetypes.DefaultGenesis().EpochDetails
	ts.Keepers.Epochstorage.SetEpochDetails(ts.Ctx, epochDetails)

	tests := []struct {
		name         string
		moniker      string
		validStake   bool
		validMoniker bool
	}{
		{"NormalMoniker", "exampleMoniker", true, true},
		{"WeirdCharsMoniker", "ビッグファームへようこそ", true, true},
		{"OversizedMoniker", "aReallyReallyReallyReallyReallyReallyReallyLongMoniker", false, false}, // too long
	}

	for it, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ts.AdvanceEpoch()

			// Note: using the same "ts" means each provider added gets a new index ("it")
			err := ts.addProviderMoniker(1, tt.moniker)
			require.Equal(t, tt.validStake, err == nil)
			providerAcct, _ := ts.GetAccount(common.PROVIDER, it)

			ts.AdvanceEpoch()

			// Get the stake entry and check the provider is staked
			stakeEntry, foundProvider, _ := ts.Keepers.Epochstorage.GetStakeEntryByAddressCurrent(ts.Ctx, ts.spec.Index, providerAcct.Addr)
			require.Equal(t, tt.validStake, foundProvider)

			// Check the assigned moniker
			if tt.validMoniker {
				require.Equal(t, tt.moniker, stakeEntry.Moniker)
			} else {
				require.NotEqual(t, tt.moniker, stakeEntry.Moniker)
			}
		})
	}
}

func TestModifyStakeProviderWithMoniker(t *testing.T) {
	ts := newTester(t)

	epochDetails := *epochstoragetypes.DefaultGenesis().EpochDetails
	ts.Keepers.Epochstorage.SetEpochDetails(ts.Ctx, epochDetails)
	ts.AdvanceEpoch()

	moniker := "exampleMoniker"
	err := ts.addProviderMoniker(1, moniker)
	require.Nil(t, err)
	ts.AdvanceEpoch()

	providerAcct, providerAddr := ts.GetAccount(common.PROVIDER, 0)

	// Get the stake entry and check the provider is staked
	stakeEntry, foundProvider, _ := ts.Keepers.Epochstorage.GetStakeEntryByAddressCurrent(ts.Ctx, ts.spec.Index, providerAcct.Addr)
	require.True(t, foundProvider)
	require.Equal(t, moniker, stakeEntry.Moniker)

	// modify moniker
	moniker = "anotherExampleMoniker"
	ts.StakeProviderExtra(providerAddr, ts.spec, testStake, nil, 0, moniker)
	ts.AdvanceEpoch()

	// Get the stake entry and check the provider is staked
	stakeEntry, foundProvider, _ = ts.Keepers.Epochstorage.GetStakeEntryByAddressCurrent(ts.Ctx, ts.spec.Index, providerAcct.Addr)
	require.True(t, foundProvider)

	require.Equal(t, moniker, stakeEntry.Moniker)
}

func TestCmdStakeProviderGeoConfigAndEnum(t *testing.T) {
	ts := newTester(t)
	ts.setupForPayments(1, 1, 1)
	_, provider := ts.AddAccount(common.PROVIDER, 50, testBalance)

	buildEndpoint := func(geoloc string) []string {
		hostip := "127.0.0.1:3351"
		apiInterface := "jsonrpc"
		return []string{hostip + "," + geoloc + "," + apiInterface}
	}

	testCases := []struct {
		name        string
		endpoints   []string
		geolocation string
		valid       bool
	}{
		// single uint geolocation config tests
		{
			name:        "Single uint geolocation - happy flow",
			endpoints:   buildEndpoint("1"),
			geolocation: "1",
			valid:       true,
		},
		{
			name:        "Single uint geolocation - endpoint geo not equal to geo",
			endpoints:   buildEndpoint("2"),
			geolocation: "1",
			valid:       false,
		},
		{
			name:        "Single uint geolocation - endpoint geo not equal to geo (geo includes endpoint geo)",
			endpoints:   buildEndpoint("1"),
			geolocation: "3",
			valid:       false,
		},
		{
			name:        "Single uint geolocation - endpoint has geo of multiple regions",
			endpoints:   buildEndpoint("3"),
			geolocation: "3",
			valid:       false,
		},
		{
			name:        "Single uint geolocation - bad endpoint geo",
			endpoints:   buildEndpoint("20555"),
			geolocation: "1",
			valid:       false,
		},

		// single string geolocation config tests
		{
			name:        "Single string geolocation - happy flow",
			endpoints:   buildEndpoint("EU"),
			geolocation: "EU",
			valid:       true,
		},
		{
			name:        "Single string geolocation - endpoint geo not equal to geo",
			endpoints:   buildEndpoint("AS"),
			geolocation: "EU",
			valid:       false,
		},
		{
			name:        "Single string geolocation - endpoint geo not equal to geo (geo includes endpoint geo)",
			endpoints:   buildEndpoint("EU"),
			geolocation: "EU,USC",
			valid:       false,
		},
		{
			name:        "Single string geolocation - bad geo",
			endpoints:   buildEndpoint("EU"),
			geolocation: "BLABLA",
			valid:       false,
		},
		{
			name:        "Single string geolocation - bad geo",
			endpoints:   buildEndpoint("BLABLA"),
			geolocation: "EU",
			valid:       false,
		},

		// multiple uint geolocation config tests
		{
			name:        "Multiple uint geolocations - happy flow",
			endpoints:   append(buildEndpoint("1"), buildEndpoint("2")...),
			geolocation: "3",
			valid:       true,
		},
		{
			name:        "Multiple uint geolocations - endpoint geo not equal to geo",
			endpoints:   append(buildEndpoint("1"), buildEndpoint("4")...),
			geolocation: "2",
			valid:       false,
		},
		{
			name:        "Multiple uint geolocations - one endpoint has multi-region geo",
			endpoints:   append(buildEndpoint("1"), buildEndpoint("3")...),
			geolocation: "2",
			valid:       false,
		},

		// multiple string geolocation config tests
		{
			name:        "Multiple string geolocations - happy flow",
			endpoints:   append(buildEndpoint("AS"), buildEndpoint("EU")...),
			geolocation: "EU,AS",
			valid:       true,
		},
		{
			name:        "Multiple string geolocations - endpoint geo not equal to geo",
			endpoints:   append(buildEndpoint("EU"), buildEndpoint("USC")...),
			geolocation: "EU,AS",
			valid:       false,
		},

		// global config tests
		{
			name:        "Global uint geolocation - happy flow",
			endpoints:   buildEndpoint("65535"),
			geolocation: "65535",
			valid:       true,
		},
		{
			name:        "Global uint geolocation - happy flow 2 - global in one endpoint",
			endpoints:   append(buildEndpoint("2"), buildEndpoint("65535")...),
			geolocation: "65535",
			valid:       true,
		},
		{
			name:        "Global uint geolocation - endpoint geo not match geo",
			endpoints:   append(buildEndpoint("2"), buildEndpoint("65535")...),
			geolocation: "7",
			valid:       false,
		},
		{
			name:        "Global string geolocation - happy flow",
			endpoints:   buildEndpoint("GL"),
			geolocation: "GL",
			valid:       true,
		},
		{
			name:        "Global string geolocation - happy flow 2 - global in one endpoint",
			endpoints:   append(buildEndpoint("EU"), buildEndpoint("GL")...),
			geolocation: "GL",
			valid:       true,
		},
		{
			name:        "Global string geolocation - endpoint geo not match geo",
			endpoints:   append(buildEndpoint("EU"), buildEndpoint("GL")...),
			geolocation: "EU,AS,USC",
			valid:       false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			endpoints, geo, err := cli.HandleEndpointsAndGeolocationArgs(tc.endpoints, tc.geolocation)
			if tc.valid {
				require.Nil(t, err)
				// adjust endpoints to match the default API interfaces and addons generated with ts
				for i := 0; i < len(endpoints); i++ {
					endpoints[i].ApiInterfaces = []string{"stub"}
					endpoints[i].Addons = []string{}
				}
				_, err = ts.TxPairingStakeProvider(provider, ts.spec.Index, ts.spec.MinStakeProvider, endpoints, uint64(geo), "prov")
				require.Nil(t, err)
			} else {
				require.NotNil(t, err)
			}
		})
	}
}

func getExtensions(names ...string) []*spectypes.Extension {
	extensions := []*spectypes.Extension{}
	for _, name := range names {
		extensions = append(extensions, &spectypes.Extension{Name: name})
	}
	return extensions
}

func TestStakeEndpoints(t *testing.T) {
	ts := newTester(t)

	epochDetails := *epochstoragetypes.DefaultGenesis().EpochDetails
	ts.Keepers.Epochstorage.SetEpochDetails(ts.Ctx, epochDetails)

	apiCollections := []*spectypes.ApiCollection{
		{
			CollectionData: spectypes.CollectionData{
				ApiInterface: "mandatory",
				InternalPath: "",
				Type:         "",
				AddOn:        "",
			},
			Enabled:    true,
			Extensions: getExtensions("ext1", "ext2", "ext3"),
		},
		{
			CollectionData: spectypes.CollectionData{
				ApiInterface: "mandatory2",
				InternalPath: "",
				Type:         "",
				AddOn:        "",
			},
			Enabled:    true,
			Extensions: getExtensions("ext2", "ext3"),
		},
		{
			CollectionData: spectypes.CollectionData{
				ApiInterface: "mandatory2",
				InternalPath: "",
				Type:         "banana",
				AddOn:        "",
			},
			Enabled:    true,
			Extensions: getExtensions("ext2"),
		},
		{
			CollectionData: spectypes.CollectionData{
				ApiInterface: "mandatory",
				InternalPath: "",
				Type:         "",
				AddOn:        "addon",
			},
			Enabled:    true,
			Extensions: getExtensions("ext2"),
		},
		{
			CollectionData: spectypes.CollectionData{
				ApiInterface: "mandatory2",
				InternalPath: "",
				Type:         "",
				AddOn:        "addon",
			},
			Enabled:    true,
			Extensions: getExtensions("ext2"),
		},
		{
			CollectionData: spectypes.CollectionData{
				ApiInterface: "mandatory",
				InternalPath: "",
				Type:         "",
				AddOn:        "unique-addon",
			},
			Enabled:    true,
			Extensions: getExtensions("ext1", "ext-unique"),
		},
		{
			CollectionData: spectypes.CollectionData{
				ApiInterface: "optional",
				InternalPath: "",
				Type:         "",
				AddOn:        "optional",
			},
			Enabled:    true,
			Extensions: getExtensions("ext2"),
		},
	}

	// will overwrite the default "mock" spec
	ts.spec.ApiCollections = apiCollections
	ts.AddSpec("mock", ts.spec)

	providerAcc, providerAddr := ts.AddAccount(common.PROVIDER, 0, testBalance)

	getEndpoint := func(
		host string,
		apiInterfaces []string,
		addons []string,
		geoloc uint64,
	) epochstoragetypes.Endpoint {
		return epochstoragetypes.Endpoint{
			IPPORT:        host,
			Geolocation:   geoloc,
			Addons:        addons,
			ApiInterfaces: apiInterfaces,
		}
	}

	getEndpointWithExt := func(
		host string,
		apiInterfaces []string,
		addons []string,
		geoloc uint64,
		extensions []string,
	) epochstoragetypes.Endpoint {
		return epochstoragetypes.Endpoint{
			IPPORT:        host,
			Geolocation:   geoloc,
			Addons:        addons,
			ApiInterfaces: apiInterfaces,
			Extensions:    extensions,
		}
	}

	type testEndpoint struct {
		name        string
		endpoints   []epochstoragetypes.Endpoint
		success     bool
		geolocation uint64
		addons      int
		extensions  int
	}
	playbook := []testEndpoint{
		{
			name:        "empty single",
			endpoints:   append([]epochstoragetypes.Endpoint{}, getEndpoint("123", []string{}, []string{}, 1)),
			success:     true,
			geolocation: 1,
		},
		{
			name:        "partial apiInterface implementation",
			endpoints:   append([]epochstoragetypes.Endpoint{}, getEndpoint("123", []string{"mandatory"}, []string{}, 1)),
			success:     false,
			geolocation: 1,
		},
		{
			name:        "explicit",
			endpoints:   append([]epochstoragetypes.Endpoint{}, getEndpoint("123", []string{"mandatory", "mandatory2"}, []string{}, 1)),
			success:     true,
			geolocation: 1,
		},
		{
			name:        "divided explicit",
			endpoints:   append([]epochstoragetypes.Endpoint{getEndpoint("123", []string{"mandatory"}, []string{}, 1)}, getEndpoint("123", []string{"mandatory2"}, []string{}, 1)),
			success:     true,
			geolocation: 1,
		},
		{
			name:        "partial in each geolocation",
			endpoints:   append([]epochstoragetypes.Endpoint{getEndpoint("123", []string{"mandatory"}, []string{}, 1)}, getEndpoint("123", []string{"mandatory2"}, []string{}, 2)),
			success:     false,
			geolocation: 3,
		},
		{
			name: "empty multi geo",
			endpoints: []epochstoragetypes.Endpoint{
				getEndpoint("123", []string{}, []string{}, 1),
				getEndpoint("123", []string{}, []string{}, 2),
			},
			success:     true,
			geolocation: 3,
		},
		{
			name: "explicit divided multi geo",
			endpoints: []epochstoragetypes.Endpoint{
				getEndpoint("123", []string{"mandatory"}, []string{}, 1),
				getEndpoint("123", []string{"mandatory2"}, []string{}, 1),
				getEndpoint("123", []string{"mandatory"}, []string{}, 2),
				getEndpoint("123", []string{"mandatory2"}, []string{}, 2),
			},
			success:     true,
			geolocation: 3,
		},
		{
			name: "explicit divided multi geo in addons split",
			endpoints: []epochstoragetypes.Endpoint{
				getEndpoint("123", []string{}, []string{"mandatory"}, 1),
				getEndpoint("123", []string{}, []string{"mandatory2"}, 1),
				getEndpoint("123", []string{}, []string{"mandatory"}, 2),
				getEndpoint("123", []string{}, []string{"mandatory2"}, 2),
			},
			success:     true,
			geolocation: 3,
		},
		{
			name: "explicit divided multi geo in addons together",
			endpoints: []epochstoragetypes.Endpoint{
				getEndpoint("123", []string{}, []string{"mandatory", "mandatory2"}, 1),
				getEndpoint("123", []string{}, []string{"mandatory", "mandatory2"}, 2),
			},
			success:     true,
			geolocation: 3,
		},
		{
			name: "empty with addon partial-geo",
			endpoints: []epochstoragetypes.Endpoint{
				getEndpoint("123", []string{}, []string{"addon"}, 1),
				getEndpoint("123", []string{}, []string{""}, 2),
			},
			success:     true,
			geolocation: 3,
			addons:      1,
		},
		{
			name: "empty with addon multi-geo",
			endpoints: []epochstoragetypes.Endpoint{
				getEndpoint("123", []string{}, []string{"addon"}, 1),
				getEndpoint("123", []string{}, []string{"addon"}, 2),
			},
			success:     true,
			geolocation: 3,
			addons:      2,
		},
		{
			name: "empty with unique addon",
			endpoints: []epochstoragetypes.Endpoint{
				getEndpoint("123", []string{}, []string{"addon", "unique-addon"}, 1),
				getEndpoint("123", []string{}, []string{"addon"}, 2),
			},
			success:     false,
			geolocation: 3,
		},
		{
			name: "explicit with unique addon partial geo",
			endpoints: []epochstoragetypes.Endpoint{
				getEndpoint("123", []string{"mandatory"}, []string{"unique-addon"}, 1),
				getEndpoint("123", []string{"mandatory2"}, []string{}, 1),
				getEndpoint("123", []string{}, []string{"addon"}, 2),
			},
			success:     true,
			geolocation: 3,
			addons:      2,
		},
		{
			name: "explicit with addon + unique addon partial geo",
			endpoints: []epochstoragetypes.Endpoint{
				getEndpoint("123", []string{"mandatory"}, []string{"addon", "unique-addon"}, 1),
				getEndpoint("123", []string{"mandatory2"}, []string{"addon"}, 1),
				getEndpoint("123", []string{}, []string{"addon"}, 2),
			},
			success:     true,
			geolocation: 3,
			addons:      4,
		},
		{
			name: "partial explicit and full emptry with addon + unique addon",
			endpoints: []epochstoragetypes.Endpoint{
				getEndpoint("123", []string{"mandatory"}, []string{"addon", "unique-addon"}, 1),
				getEndpoint("123", []string{}, []string{"addon"}, 1),
			},
			success:     true,
			geolocation: 1,
			addons:      3,
		},
		{
			name: "explicit + optional",
			endpoints: []epochstoragetypes.Endpoint{
				getEndpoint("123", []string{}, []string{"mandatory", "mandatory2", "optional"}, 1),
			},
			success:     true,
			geolocation: 1,
		},
		{
			name: "empty + explicit optional",
			endpoints: []epochstoragetypes.Endpoint{
				getEndpoint("123", []string{}, []string{}, 1),
				getEndpoint("123", []string{"optional"}, []string{}, 1),
			},
			success:     true,
			geolocation: 1,
		},
		{
			name: "empty + explicit optional in addon",
			endpoints: []epochstoragetypes.Endpoint{
				getEndpoint("123", []string{}, []string{}, 1),
				getEndpoint("123", []string{}, []string{"optional"}, 1),
			},
			success:     true,
			geolocation: 1,
		},
		{
			name: "empty + explicit optional + optional addon",
			endpoints: []epochstoragetypes.Endpoint{
				getEndpoint("123", []string{}, []string{}, 1),
				getEndpoint("123", []string{"optional"}, []string{"optional"}, 1),
			},
			success:     true,
			geolocation: 1,
		},
		{
			name: "explicit optional",
			endpoints: []epochstoragetypes.Endpoint{
				getEndpoint("123", []string{"optional"}, []string{"optional"}, 1),
			},
			success:     false,
			geolocation: 1,
		},
		{
			name: "full partial geo",
			endpoints: []epochstoragetypes.Endpoint{
				getEndpoint("123", []string{}, []string{"addon"}, 1),
				getEndpoint("123", []string{"mandatory"}, []string{"unique-addon"}, 1),
				getEndpoint("123", []string{"optional"}, []string{}, 1),
				getEndpoint("123", []string{}, []string{}, 2),
			},
			success:     true,
			geolocation: 3,
			addons:      2,
		},
		{
			name: "full multi geo",
			endpoints: []epochstoragetypes.Endpoint{
				getEndpoint("123", []string{}, []string{"addon"}, 1),
				getEndpoint("123", []string{"mandatory"}, []string{"unique-addon"}, 1),
				getEndpoint("123", []string{"optional"}, []string{}, 1),
				getEndpoint("123", []string{}, []string{"addon"}, 2),
				getEndpoint("123", []string{"mandatory"}, []string{"unique-addon"}, 2),
				getEndpoint("123", []string{"optional"}, []string{}, 2),
			},
			success:     true,
			geolocation: 3,
			addons:      4,
		},
		{
			name: "mandatory with extension - multi geo",
			endpoints: []epochstoragetypes.Endpoint{
				getEndpointWithExt("123", []string{}, []string{}, 1, []string{"ext2"}),
				getEndpointWithExt("123", []string{}, []string{}, 2, []string{"ext2"}),
			},
			success:     true,
			geolocation: 3,
			extensions:  2,
		},
		{
			name: "mandatory as extension in addon - multi geo",
			endpoints: []epochstoragetypes.Endpoint{
				getEndpointWithExt("123", []string{}, []string{"ext2"}, 1, []string{}),
				getEndpointWithExt("123", []string{}, []string{"ext2"}, 2, []string{}),
			},
			success:     true,
			geolocation: 3,
			extensions:  2,
		},
		{
			name: "mandatory with two extensions - multi geo",
			endpoints: []epochstoragetypes.Endpoint{
				getEndpointWithExt("123", []string{}, []string{}, 1, []string{"ext3", "ext2"}),
				getEndpointWithExt("123", []string{}, []string{}, 2, []string{"ext3", "ext2"}),
			},
			success:     true,
			geolocation: 3,
			extensions:  4,
		},
		{
			name: "invalid ext",
			endpoints: []epochstoragetypes.Endpoint{
				getEndpointWithExt("123", []string{}, []string{}, 1, []string{"invalid"}),
			},
			success:     false,
			geolocation: 1,
		},
		{
			name: "invalid ext two",
			endpoints: []epochstoragetypes.Endpoint{
				getEndpointWithExt("123", []string{}, []string{}, 1, []string{"ext1", "invalid"}),
			},
			success:     false,
			geolocation: 1,
		},
		{
			name: "mandatory with unique extension",
			endpoints: []epochstoragetypes.Endpoint{
				getEndpointWithExt("123", []string{"mandatory"}, []string{}, 1, []string{"ext1"}),
				getEndpointWithExt("123", []string{"mandatory2"}, []string{}, 1, []string{}),
			},
			success:     true,
			geolocation: 1,
			extensions:  1,
		},
		{
			name: "mandatory with all extensions",
			endpoints: []epochstoragetypes.Endpoint{
				getEndpointWithExt("123", []string{"mandatory"}, []string{}, 1, []string{"ext1"}),
				getEndpointWithExt("123", []string{}, []string{}, 1, []string{"ext3", "ext2"}),
			},
			success:     true,
			geolocation: 1,
			extensions:  3,
		},
		{
			name: "mandatory with addon and extension - multi geo",
			endpoints: []epochstoragetypes.Endpoint{
				getEndpointWithExt("123", []string{}, []string{"addon"}, 1, []string{"ext2"}),
				getEndpointWithExt("123", []string{}, []string{"addon"}, 2, []string{"ext2"}),
			},
			success:     true,
			geolocation: 3,
			addons:      2,
			extensions:  2,
		},
		{
			name: "mandatory with addon and extension as addon - multi geo",
			endpoints: []epochstoragetypes.Endpoint{
				getEndpointWithExt("123", []string{}, []string{"addon", "ext2"}, 1, []string{}),
				getEndpointWithExt("123", []string{}, []string{"addon", "ext2"}, 2, []string{}),
			},
			success:     true,
			geolocation: 3,
			addons:      2,
			extensions:  2,
		},
		{
			name: "mandatory unique addon with unique ext",
			endpoints: []epochstoragetypes.Endpoint{
				getEndpointWithExt("123", []string{}, []string{}, 1, []string{}),
				getEndpointWithExt("123", []string{"mandatory"}, []string{"unique-addon"}, 1, []string{"ext-unique"}),
			},
			success:     false, // we fail this because extensions that don't exist in the parent apiCollection can't work
			geolocation: 1,
		},
		{
			name: "explicit + optional with extension",
			endpoints: []epochstoragetypes.Endpoint{
				getEndpointWithExt("123", []string{}, []string{"mandatory", "mandatory2", "optional"}, 1, []string{"ext2"}),
			},
			success:     true,
			geolocation: 1,
			extensions:  1,
		},
	}

	amount := common.NewCoin(testStake)

	for _, play := range playbook {
		t.Run(play.name, func(t *testing.T) {
			_, err := ts.TxPairingStakeProvider(providerAddr, ts.spec.Index, amount, play.endpoints, play.geolocation, "prov")
			if play.success {
				require.NoError(t, err)

				providerEntry, found, _ := ts.Keepers.Epochstorage.GetStakeEntryByAddressCurrent(ts.Ctx, ts.spec.Index, providerAcc.Addr)
				require.True(t, found)
				addons := 0
				extensions := 0
				for _, endpoint := range providerEntry.Endpoints {
					for _, addon := range endpoint.Addons {
						if addon != "" {
							addons++
						}
					}
					for _, extension := range endpoint.Extensions {
						if extension != "" {
							extensions++
						}
					}
				}
				require.Equal(t, play.addons, addons, providerEntry)
				require.Equal(t, play.extensions, extensions)
			} else {
				require.Error(t, err)
			}
		})
	}
}
