package keeper_test

import (
	"testing"
	"time"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/testutil/common"
	epochstoragetypes "github.com/lavanet/lava/x/epochstorage/types"
	"github.com/lavanet/lava/x/pairing/client/cli"
	"github.com/lavanet/lava/x/pairing/types"
	spectypes "github.com/lavanet/lava/x/spec/types"
	"github.com/stretchr/testify/require"
)

// Test that the optional moniker argument in StakeProvider doesn't break anything
func TestStakeProviderWithMoniker(t *testing.T) {
	ts := newTester(t)

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
			require.Equal(t, tt.validStake, err == nil, err)
			_, provider := ts.GetAccount(common.PROVIDER, it)

			ts.AdvanceEpoch()

			// Get the stake entry and check the provider is staked
			stakeEntry, foundProvider := ts.Keepers.Epochstorage.GetStakeEntryByAddressCurrent(ts.Ctx, ts.spec.Index, provider)
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

	ts.AdvanceEpoch()

	moniker := "exampleMoniker"
	err := ts.addProviderMoniker(1, moniker)
	require.NoError(t, err)
	ts.AdvanceEpoch()

	providerAcct, providerAddr := ts.GetAccount(common.PROVIDER, 0)

	// Get the stake entry and check the provider is staked
	stakeEntry, foundProvider := ts.Keepers.Epochstorage.GetStakeEntryByAddressCurrent(ts.Ctx, ts.spec.Index, providerAddr)
	require.True(t, foundProvider)
	require.Equal(t, moniker, stakeEntry.Moniker)

	// modify moniker
	moniker = "anotherExampleMoniker"
	err = ts.StakeProviderExtra(providerAcct.GetVaultAddr(), providerAddr, ts.spec, testStake, nil, 0, moniker)
	require.NoError(t, err)
	ts.AdvanceEpoch()

	// Get the stake entry and check the provider is staked
	stakeEntry, foundProvider = ts.Keepers.Epochstorage.GetStakeEntryByAddressCurrent(ts.Ctx, ts.spec.Index, providerAddr)
	require.True(t, foundProvider)

	require.Equal(t, moniker, stakeEntry.Moniker)
}

func TestCmdStakeProviderGeoConfigAndEnum(t *testing.T) {
	ts := newTester(t)
	ts.setupForPayments(1, 1, 1)
	acc, provider := ts.AddAccount(common.PROVIDER, 50, testBalance)

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
				require.NoError(t, err)
				// adjust endpoints to match the default API interfaces and addons generated with ts
				for i := 0; i < len(endpoints); i++ {
					endpoints[i].ApiInterfaces = []string{"stub"}
					endpoints[i].Addons = []string{}
				}
				_, err = ts.TxPairingStakeProvider(provider, acc.GetVaultAddr(), ts.spec.Index, ts.spec.MinStakeProvider, endpoints, geo, "prov")
				require.NoError(t, err)
			} else {
				require.Error(t, err)
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
		geoloc int32,
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
		geoloc int32,
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
		geolocation int32
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

	amount := common.NewCoin(ts.Keepers.StakingKeeper.BondDenom(ts.Ctx), testStake)

	for _, play := range playbook {
		t.Run(play.name, func(t *testing.T) {
			_, err := ts.TxPairingStakeProvider(providerAddr, providerAcc.GetVaultAddr(), ts.spec.Index, amount, play.endpoints, play.geolocation, "prov")
			if play.success {
				require.NoError(t, err)

				providerEntry, found := ts.Keepers.Epochstorage.GetStakeEntryByAddressCurrent(ts.Ctx, ts.spec.Index, providerAddr)
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

// TestStakeProviderLimits tests the staking limits
// Scenarios:
// 1. provider tries to stake below min self delegation -> stake TX should fail
// 2. provider stakes above min self delegation but below the spec's min stake -> stake should succeed but provider should be frozen
// 3. provider stakes above the spec's min stake -> stake should succeed and provider is not frozen
func TestStakeProviderLimits(t *testing.T) {
	// set MinSelfDelegation = 100, MinStakeProvider = 200
	ts := newTester(t)
	minSelfDelegation := ts.Keepers.Dualstaking.MinSelfDelegation(ts.Ctx)
	ts.spec.MinStakeProvider = minSelfDelegation.AddAmount(math.NewInt(100))
	ts.Keepers.Spec.SetSpec(ts.Ctx, ts.spec)
	ts.AdvanceEpoch()

	type testCase struct {
		name     string
		stake    int64
		isStaked bool
		isFrozen bool
	}
	testCases := []testCase{
		{"below min self delegation", minSelfDelegation.Amount.Int64() - 1, false, false},
		{"above min self delegation and below min provider stake", minSelfDelegation.Amount.Int64() + 1, true, true},
		{"above min provider stake", ts.spec.MinStakeProvider.Amount.Int64() + 1, true, false},
	}

	for it, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			providerAcct, addr := ts.AddAccount(common.PROVIDER, it+1, tt.stake)
			err := ts.StakeProviderExtra(providerAcct.GetVaultAddr(), addr, ts.spec, tt.stake, nil, 0, "")
			if !tt.isStaked {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			stakeEntry, found := ts.Keepers.Epochstorage.GetStakeEntryByAddressCurrent(ts.Ctx, ts.spec.Index, addr)
			require.True(t, found)
			require.Equal(t, tt.isFrozen, stakeEntry.IsFrozen())
		})
	}
}

// TestUnfreezeWithDelegations checks the following scenario:
// a provider stakes below the spec's min stake, so it's frozen (and can't unfreeze due to small stake)
// Then, delegators add to its effective stake to be above min stake, provider is still frozen but now
// can unfreeze
func TestUnfreezeWithDelegations(t *testing.T) {
	// set MinSelfDelegation = 100, MinStakeProvider = 200
	ts := newTester(t)
	minSelfDelegation := ts.Keepers.Dualstaking.MinSelfDelegation(ts.Ctx)
	ts.spec.MinStakeProvider = minSelfDelegation.AddAmount(math.NewInt(100))
	ts.Keepers.Spec.SetSpec(ts.Ctx, ts.spec)
	ts.AdvanceEpoch()

	// stake minSelfDelegation+1 -> operator staked but frozen
	providerAcc, operator := ts.AddAccount(common.PROVIDER, 1, minSelfDelegation.Amount.Int64()+1)
	err := ts.StakeProviderExtra(providerAcc.GetVaultAddr(), operator, ts.spec, minSelfDelegation.Amount.Int64()+1, nil, 0, "")
	require.NoError(t, err)
	stakeEntry, found := ts.Keepers.Epochstorage.GetStakeEntryByAddressCurrent(ts.Ctx, ts.spec.Index, operator)
	require.True(t, found)
	require.True(t, stakeEntry.IsFrozen())
	require.Equal(t, minSelfDelegation.Amount.AddRaw(1), stakeEntry.EffectiveStake())

	// try to unfreeze -> should fail
	_, err = ts.TxPairingUnfreezeProvider(operator, ts.spec.Index)
	require.Error(t, err)

	// increase delegation limit of stake entry from 0 to MinStakeProvider + 100
	stakeEntry, found = ts.Keepers.Epochstorage.GetStakeEntryByAddressCurrent(ts.Ctx, ts.spec.Index, operator)
	require.True(t, found)
	stakeEntry.DelegateLimit = ts.spec.MinStakeProvider.AddAmount(math.NewInt(100))
	ts.Keepers.Epochstorage.ModifyStakeEntryCurrent(ts.Ctx, ts.spec.Index, stakeEntry)
	ts.AdvanceEpoch()

	// add delegator and delegate to provider so its effective stake is MinStakeProvider+MinSelfDelegation+1
	// provider should still be frozen
	_, consumer := ts.AddAccount(common.CONSUMER, 1, testBalance)
	_, err = ts.TxDualstakingDelegate(consumer, operator, ts.spec.Index, ts.spec.MinStakeProvider)
	require.NoError(t, err)
	ts.AdvanceEpoch() // apply delegation
	stakeEntry, found = ts.Keepers.Epochstorage.GetStakeEntryByAddressCurrent(ts.Ctx, ts.spec.Index, operator)
	require.True(t, found)
	require.True(t, stakeEntry.IsFrozen())
	require.Equal(t, ts.spec.MinStakeProvider.Add(minSelfDelegation).Amount.AddRaw(1), stakeEntry.EffectiveStake())

	// try to unfreeze -> should succeed
	_, err = ts.TxPairingUnfreezeProvider(operator, ts.spec.Index)
	require.NoError(t, err)
}

// tests the commission and delegation limit changes
// - changes without delegations (no limitations)
// - first change within the 1% limitation
// - try to change before 24H
// - change after 24H within the expected values
// - change after 24H outside of allowed values
func TestCommisionChange(t *testing.T) {
	// set MinSelfDelegation = 100, MinStakeProvider = 200
	ts := newTester(t)
	minSelfDelegation := ts.Keepers.Dualstaking.MinSelfDelegation(ts.Ctx)
	ts.spec.MinStakeProvider = minSelfDelegation.AddAmount(math.NewInt(100))
	ts.Keepers.Spec.SetSpec(ts.Ctx, ts.spec)
	ts.AdvanceEpoch()

	_, provider := ts.AddAccount(common.PROVIDER, 1, ts.spec.MinStakeProvider.Amount.Int64())
	_, err := ts.TxPairingStakeProviderFull(provider, provider, ts.spec.Index, ts.spec.MinStakeProvider, nil, 0, "", 50, 100)
	require.NoError(t, err)

	// there are no delegations, can change as much as we want
	_, err = ts.TxPairingStakeProviderFull(provider, provider, ts.spec.Index, ts.spec.MinStakeProvider, nil, 0, "", 55, 120)
	require.NoError(t, err)

	_, err = ts.TxPairingStakeProviderFull(provider, provider, ts.spec.Index, ts.spec.MinStakeProvider, nil, 0, "", 60, 140)
	require.NoError(t, err)

	// add delegator and delegate to provider
	_, consumer := ts.AddAccount(common.CONSUMER, 1, testBalance)
	_, err = ts.TxDualstakingDelegate(consumer, provider, ts.spec.Index, ts.spec.MinStakeProvider)
	require.NoError(t, err)
	ts.AdvanceEpoch()               // apply delegation
	ts.AdvanceBlock(time.Hour * 25) // advance time to allow changes

	// now changes are limited
	_, err = ts.TxPairingStakeProviderFull(provider, provider, ts.spec.Index, ts.spec.MinStakeProvider, nil, 0, "", 61, 139)
	require.NoError(t, err)

	_, err = ts.TxPairingStakeProviderFull(provider, provider, ts.spec.Index, ts.spec.MinStakeProvider, nil, 0, "", 62, 138)
	require.Error(t, err)

	ts.AdvanceBlock(time.Hour * 25)

	_, err = ts.TxPairingStakeProviderFull(provider, provider, ts.spec.Index, ts.spec.MinStakeProvider, nil, 0, "", 62, 138)
	require.NoError(t, err)

	ts.AdvanceBlock(time.Hour * 25)

	_, err = ts.TxPairingStakeProviderFull(provider, provider, ts.spec.Index, ts.spec.MinStakeProvider, nil, 0, "", 68, 100)
	require.Error(t, err)
}

// TestVaultOperatorNewStakeEntry tests that staking (or "self delegation") is actually the provider's vault
// delegating to the provider's operator address.
// For each scenario, we'll check the vault's balance, the stake entry's stake/delegate total fields and the delegations
// fixation store.
// Scenarios:
// 1. stake new entry with operator = vault
// 2. stake new entry with different addresses for operator and vault
// 3. stake new entry with operator -> should fail since operator is already registered to a stake entry
// 4. stake new entry with operator on different chain -> should work
// 5. stake to an existing entry with vault -> should succeed
// 6. stake to an existing entry with operator -> should fail
func TestVaultOperatorNewStakeEntry(t *testing.T) {
	ts := newTester(t)

	p1Acc, _ := ts.AddAccount(common.PROVIDER, 0, testBalance)
	p2Acc, _ := ts.AddAccount(common.PROVIDER, 1, testBalance)

	spec1 := ts.spec
	spec1.Index = "spec1"
	ts.AddSpec("spec1", spec1)

	tests := []struct {
		name     string
		vault    sdk.AccAddress
		operator sdk.AccAddress
		spec     spectypes.Spec
		valid    bool
	}{
		{"stake operator = vault", p1Acc.Addr, p1Acc.Addr, ts.spec, true},
		{"stake operator != vault", p2Acc.Vault.Addr, p2Acc.Addr, ts.spec, true},
		{"stake existing operator", p1Acc.Vault.Addr, p1Acc.Addr, ts.spec, false},
		{"stake existing operator different chain", p1Acc.Vault.Addr, p1Acc.Addr, spec1, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			vaultBefore := ts.GetBalance(tt.vault)
			operatorBefore := ts.GetBalance(tt.operator)
			err := ts.StakeProviderExtra(tt.vault.String(), tt.operator.String(), tt.spec, testStake, []epochstoragetypes.Endpoint{{Geolocation: 1}}, 1, "test")
			vaultAfter := ts.GetBalance(tt.vault)
			operatorAfter := ts.GetBalance(tt.operator)

			ts.AdvanceEpoch()

			if !tt.valid {
				require.Error(t, err)

				// balance
				require.Equal(t, vaultBefore, vaultAfter)
				require.Equal(t, operatorBefore, operatorAfter)

				// stake entry
				_, found := ts.Keepers.Epochstorage.GetStakeEntryByAddressCurrent(ts.Ctx, tt.spec.Index, tt.operator.String())
				require.True(t, found) // should be found because operator is registered in a vaild stake entry

				// delegations
				res, err := ts.QueryDualstakingDelegatorProviders(tt.vault.String(), false)
				require.NoError(t, err)
				require.Len(t, res.Delegations, 0)
			} else {
				require.NoError(t, err)

				// balance
				require.Equal(t, vaultBefore-testStake, vaultAfter)
				if tt.operator.Equals(tt.vault) {
					require.Equal(t, operatorBefore-testStake, operatorAfter)
				} else {
					require.Equal(t, operatorBefore, operatorAfter)
				}

				// stake entry
				stakeEntry, found := ts.Keepers.Epochstorage.GetStakeEntryByAddressCurrent(ts.Ctx, tt.spec.Index, tt.operator.String())
				require.True(t, found)
				require.Equal(t, testStake, stakeEntry.Stake.Amount.Int64())
				require.Equal(t, int64(0), stakeEntry.DelegateTotal.Amount.Int64())

				// delegations
				res, err := ts.QueryDualstakingDelegatorProviders(tt.vault.String(), false)
				require.NoError(t, err)
				require.Len(t, res.Delegations, 1)
				require.Equal(t, tt.operator.String(), res.Delegations[0].Provider)
				require.Equal(t, testStake, res.Delegations[0].Amount.Amount.Int64())
			}
		})
	}
}

// TestVaultOperatorExistingStakeEntry tests that staking (or "self delegation") to an existing stake entry
// only works when the creator is the vault and not the operator.
// For each scenario, we'll check the vault's balance, the stake entry's stake/delegate total fields and the delegations
// fixation store.
// Scenarios:
// 1. stake with vault to an existing stake entry -> should work
// 2. try with operator -> should fail
func TestVaultOperatorExistingStakeEntry(t *testing.T) {
	ts := newTester(t)

	p1Acc, _ := ts.AddAccount(common.PROVIDER, 0, testBalance)
	err := ts.StakeProviderExtra(p1Acc.GetVaultAddr(), p1Acc.Addr.String(), ts.spec, testStake, []epochstoragetypes.Endpoint{{Geolocation: 1}}, 1, "test")
	require.NoError(t, err)

	tests := []struct {
		name     string
		vault    sdk.AccAddress
		operator sdk.AccAddress
		spec     spectypes.Spec
		valid    bool
	}{
		{"stake with vault", p1Acc.Vault.Addr, p1Acc.Addr, ts.spec, true},
		{"stake with operator", p1Acc.Addr, p1Acc.Addr, ts.spec, false},
	}

	for i, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			beforeVault := ts.GetBalance(tt.vault)
			beforeOperator := ts.GetBalance(tt.operator)
			fundsToStake := testStake + (100 * int64(i+1)) // funds to stake. needs to change each iteration for the checks below
			err := ts.StakeProviderExtra(tt.vault.String(), tt.operator.String(), tt.spec, fundsToStake, []epochstoragetypes.Endpoint{{Geolocation: 1}}, 1, "test")
			afterVault := ts.GetBalance(tt.vault)
			afterOperator := ts.GetBalance(tt.operator)

			ts.AdvanceEpoch()

			if !tt.valid {
				require.Error(t, err)

				// balance
				require.Equal(t, beforeVault, afterVault)
				require.Equal(t, beforeOperator, afterOperator)

				// stake entry
				_, found := ts.Keepers.Epochstorage.GetStakeEntryByAddressCurrent(ts.Ctx, tt.spec.Index, tt.operator.String())
				require.True(t, found) // should be found because operator is registered in a vaild stake entry

				// delegations
				res, err := ts.QueryDualstakingDelegatorProviders(tt.vault.String(), false)
				require.NoError(t, err)
				require.Len(t, res.Delegations, 0)
			} else {
				require.NoError(t, err)

				// balance
				require.Equal(t, beforeVault-100, afterVault)
				if tt.operator.Equals(tt.vault) {
					require.Equal(t, beforeOperator-100, afterOperator)
				} else {
					require.Equal(t, beforeOperator, afterOperator)
				}

				// stake entry
				stakeEntry, found := ts.Keepers.Epochstorage.GetStakeEntryByAddressCurrent(ts.Ctx, tt.spec.Index, tt.operator.String())
				require.True(t, found)
				require.Equal(t, testStake+100, stakeEntry.Stake.Amount.Int64())
				require.Equal(t, int64(0), stakeEntry.DelegateTotal.Amount.Int64())

				// delegations
				res, err := ts.QueryDualstakingDelegatorProviders(tt.vault.String(), false)
				require.NoError(t, err)
				require.Len(t, res.Delegations, 1)
				require.Equal(t, tt.operator.String(), res.Delegations[0].Provider)
				require.Equal(t, testStake+100, res.Delegations[0].Amount.Amount.Int64())
			}
		})
	}
}

// TestVaultOperatorModifyStakeEntry tests that the operator can change non-stake related properties of the stake entry
// and the vault address can change anything
// Scenarios:
// 1. operator modifies all non-stake related properties -> should work
// 2. operator modifies all stake related properties -> should fail
// 3. vault can change anything in the stake entry
func TestVaultOperatorModifyStakeEntry(t *testing.T) {
	ts := newTester(t)
	ts.setupForPayments(1, 1, 0)
	acc, _ := ts.GetAccount(common.PROVIDER, 0)
	_, dummy := ts.GetAccount(common.CONSUMER, 0)

	operator := acc.Addr.String()
	vault := acc.GetVaultAddr()
	valAcc, _ := ts.GetAccount(common.VALIDATOR, 0)

	stakeEntry, found := ts.Keepers.Epochstorage.GetStakeEntryByAddressCurrent(ts.Ctx, ts.spec.Index, acc.Addr.String())
	require.True(t, found)

	// consts for stake entry changes
	const (
		STAKE = iota + 1
		ENDPOINTS_GEOLOCATION
		MONIKER
		DELEGATE_LIMIT
		DELEGATE_COMMISSION
		OPERATOR
	)

	tests := []struct {
		name        string
		creator     string
		stakeChange int
		valid       bool
	}{
		// vault can change anything
		{"stake change vault", vault, STAKE, true},
		{"endpoints_geo change vault", vault, ENDPOINTS_GEOLOCATION, true},
		{"moniker change vault", vault, MONIKER, true},
		{"delegate total change vault", vault, DELEGATE_LIMIT, true},
		{"delegate commission change vault", vault, DELEGATE_COMMISSION, true},
		{"operator change vault", vault, OPERATOR, true},

		// operator can't change stake/delegation related properties
		{"stake change operator", operator, STAKE, false},
		{"endpoints_geo change operator", operator, ENDPOINTS_GEOLOCATION, true},
		{"moniker change operator", operator, MONIKER, true},
		{"delegate total change operator", operator, DELEGATE_LIMIT, false},
		{"delegate commission change operator", operator, DELEGATE_COMMISSION, false},
		{"operator change operator", operator, OPERATOR, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msg := types.MsgStakeProvider{
				Creator:            tt.creator,
				Validator:          sdk.ValAddress(valAcc.Addr).String(),
				ChainID:            stakeEntry.Chain,
				Amount:             stakeEntry.Stake,
				Geolocation:        stakeEntry.Geolocation,
				Endpoints:          stakeEntry.Endpoints,
				Moniker:            stakeEntry.Moniker,
				DelegateLimit:      stakeEntry.DelegateLimit,
				DelegateCommission: stakeEntry.DelegateCommission,
				Operator:           stakeEntry.Operator,
			}

			switch tt.stakeChange {
			case STAKE:
				msg.Amount = msg.Amount.AddAmount(math.NewInt(100))
			case ENDPOINTS_GEOLOCATION:
				msg.Endpoints = []epochstoragetypes.Endpoint{{Geolocation: 2}}
				msg.Geolocation = 2
			case MONIKER:
				msg.Moniker = "test2"
			case DELEGATE_LIMIT:
				msg.DelegateLimit = msg.DelegateLimit.AddAmount(math.NewInt(100))
			case DELEGATE_COMMISSION:
				msg.DelegateCommission -= 10
			case OPERATOR:
				operator = dummy
			}

			_, err := ts.TxPairingStakeProviderFull(
				msg.Creator,
				msg.Operator,
				msg.ChainID,
				msg.Amount,
				msg.Endpoints,
				msg.Geolocation,
				msg.Moniker,
				msg.DelegateCommission,
				msg.DelegateLimit.Amount.Uint64(),
			)
			if tt.valid {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
			}

			// reset stake entry
			ts.Keepers.Epochstorage.ModifyStakeEntryCurrent(ts.Ctx, ts.spec.Index, stakeEntry)
			ts.AdvanceEpoch()
		})
	}
}
