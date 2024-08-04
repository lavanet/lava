package keeper_test

import (
	"testing"
	"time"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	"github.com/lavanet/lava/v2/testutil/common"
	epochstoragetypes "github.com/lavanet/lava/v2/x/epochstorage/types"
	"github.com/lavanet/lava/v2/x/pairing/client/cli"
	"github.com/lavanet/lava/v2/x/pairing/types"
	spectypes "github.com/lavanet/lava/v2/x/spec/types"
	"github.com/stretchr/testify/require"
)

func TestModifyStakeProviderWithDescription(t *testing.T) {
	ts := newTester(t)

	ts.AdvanceEpoch()

	err := ts.addProvider(1)
	require.NoError(t, err)
	ts.AdvanceEpoch()

	providerAcct, providerAddr := ts.GetAccount(common.PROVIDER, 0)

	// Get the stake entry and check the provider is staked
	stakeEntry, foundProvider := ts.Keepers.Epochstorage.GetStakeEntryByAddressCurrent(ts.Ctx, ts.spec.Index, providerAddr)
	require.True(t, foundProvider)
	require.True(t, stakeEntry.Description.Equal(common.MockDescription()))

	// modify description
	dNew := stakingtypes.NewDescription("bla", "blan", "bal", "lala", "asdasd")
	err = ts.StakeProviderExtra(providerAcct.GetVaultAddr(), providerAddr, ts.spec, testStake, nil, 0, dNew.Moniker, dNew.Identity, dNew.Website, dNew.SecurityContact, dNew.Details)
	require.NoError(t, err)
	ts.AdvanceEpoch()

	// Get the stake entry and check the provider is staked
	stakeEntry, foundProvider = ts.Keepers.Epochstorage.GetStakeEntryByAddressCurrent(ts.Ctx, ts.spec.Index, providerAddr)
	require.True(t, foundProvider)
	require.True(t, stakeEntry.Description.Equal(dNew))
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
				_, err = ts.TxPairingStakeProvider(provider, acc.GetVaultAddr(), ts.spec.Index, ts.spec.MinStakeProvider, endpoints, geo, common.MockDescription())
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
			_, err := ts.TxPairingStakeProvider(providerAddr, providerAcc.GetVaultAddr(), ts.spec.Index, amount, play.endpoints, play.geolocation, common.MockDescription())
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
			err := ts.StakeProviderExtra(providerAcct.GetVaultAddr(), addr, ts.spec, tt.stake, nil, 0, "", "", "", "", "")
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

	// stake minSelfDelegation+1 -> provider staked but frozen
	providerAcc, provider := ts.AddAccount(common.PROVIDER, 1, minSelfDelegation.Amount.Int64()+1)
	err := ts.StakeProviderExtra(providerAcc.GetVaultAddr(), provider, ts.spec, minSelfDelegation.Amount.Int64()+1, nil, 0, "", "", "", "", "")
	require.NoError(t, err)
	stakeEntry, found := ts.Keepers.Epochstorage.GetStakeEntryByAddressCurrent(ts.Ctx, ts.spec.Index, provider)
	require.True(t, found)
	require.True(t, stakeEntry.IsFrozen())
	require.Equal(t, minSelfDelegation.Amount.AddRaw(1), stakeEntry.EffectiveStake())

	// try to unfreeze -> should fail
	_, err = ts.TxPairingUnfreezeProvider(provider, ts.spec.Index)
	require.Error(t, err)

	// increase delegation limit of stake entry from 0 to MinStakeProvider + 100
	stakeEntry, found = ts.Keepers.Epochstorage.GetStakeEntryByAddressCurrent(ts.Ctx, ts.spec.Index, provider)
	require.True(t, found)
	stakeEntry.DelegateLimit = ts.spec.MinStakeProvider.AddAmount(math.NewInt(100))
	ts.Keepers.Epochstorage.ModifyStakeEntryCurrent(ts.Ctx, ts.spec.Index, stakeEntry)
	ts.AdvanceEpoch()

	// add delegator and delegate to provider so its effective stake is MinStakeProvider+MinSelfDelegation+1
	// provider should still be frozen
	_, consumer := ts.AddAccount(common.CONSUMER, 1, testBalance)
	_, err = ts.TxDualstakingDelegate(consumer, provider, ts.spec.Index, ts.spec.MinStakeProvider)
	require.NoError(t, err)
	ts.AdvanceEpoch() // apply delegation
	stakeEntry, found = ts.Keepers.Epochstorage.GetStakeEntryByAddressCurrent(ts.Ctx, ts.spec.Index, provider)
	require.True(t, found)
	require.True(t, stakeEntry.IsFrozen())
	require.Equal(t, ts.spec.MinStakeProvider.Add(minSelfDelegation).Amount.AddRaw(1), stakeEntry.EffectiveStake())

	// try to unfreeze -> should succeed
	_, err = ts.TxPairingUnfreezeProvider(provider, ts.spec.Index)
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
	_, err := ts.TxPairingStakeProviderFull(provider, provider, ts.spec.Index, ts.spec.MinStakeProvider, nil, 0, 50, 100, "", "", "", "", "")
	require.NoError(t, err)

	// there are no delegations, can change as much as we want
	_, err = ts.TxPairingStakeProviderFull(provider, provider, ts.spec.Index, ts.spec.MinStakeProvider, nil, 0, 55, 120, "", "", "", "", "")
	require.NoError(t, err)

	_, err = ts.TxPairingStakeProviderFull(provider, provider, ts.spec.Index, ts.spec.MinStakeProvider, nil, 0, 60, 140, "", "", "", "", "")
	require.NoError(t, err)

	// add delegator and delegate to provider
	_, consumer := ts.AddAccount(common.CONSUMER, 1, testBalance)
	_, err = ts.TxDualstakingDelegate(consumer, provider, ts.spec.Index, ts.spec.MinStakeProvider)
	require.NoError(t, err)
	ts.AdvanceEpoch()               // apply delegation
	ts.AdvanceBlock(time.Hour * 25) // advance time to allow changes

	// now changes are limited
	_, err = ts.TxPairingStakeProviderFull(provider, provider, ts.spec.Index, ts.spec.MinStakeProvider, nil, 0, 61, 139, "", "", "", "", "")
	require.NoError(t, err)

	// same values, should pass
	_, err = ts.TxPairingStakeProviderFull(provider, provider, ts.spec.Index, ts.spec.MinStakeProvider, nil, 0, 61, 139, "", "", "", "", "")
	require.NoError(t, err)

	_, err = ts.TxPairingStakeProviderFull(provider, provider, ts.spec.Index, ts.spec.MinStakeProvider, nil, 0, 62, 138, "", "", "", "", "")
	require.Error(t, err)

	ts.AdvanceBlock(time.Hour * 25)

	_, err = ts.TxPairingStakeProviderFull(provider, provider, ts.spec.Index, ts.spec.MinStakeProvider, nil, 0, 62, 138, "", "", "", "", "")
	require.NoError(t, err)

	ts.AdvanceBlock(time.Hour * 25)

	_, err = ts.TxPairingStakeProviderFull(provider, provider, ts.spec.Index, ts.spec.MinStakeProvider, nil, 0, 68, 100, "", "", "", "", "")
	require.Error(t, err)
}

// TestVaultProviderNewStakeEntry tests that staking (or "self delegation") is actually the provider's vault
// delegating to the provider's address.
// For each scenario, we'll check the vault's balance, the stake entry's stake/delegate total fields and the delegations
// fixation store.
// Scenarios:
// 1. stake new entry with provider = vault
// 2. stake new entry with different addresses for provider and vault
// 3. stake new entry with provider -> should fail since provider is already registered to a stake entry
// 4. stake new entry with provider on different chain -> should work
// 5. stake to an existing entry with vault -> should succeed
// 6. stake to an existing entry with provider -> should fail
func TestVaultProviderNewStakeEntry(t *testing.T) {
	ts := newTester(t)

	p1Acc, _ := ts.AddAccount(common.PROVIDER, 0, testBalance)
	p2Acc, _ := ts.AddAccount(common.PROVIDER, 1, testBalance)

	spec1 := ts.spec
	spec1.Index = "spec1"
	ts.AddSpec("spec1", spec1)

	tests := []struct {
		name     string
		vault    sdk.AccAddress
		provider sdk.AccAddress
		spec     spectypes.Spec
		valid    bool
	}{
		{"stake provider = vault", p1Acc.Addr, p1Acc.Addr, ts.spec, true},
		{"stake provider != vault", p2Acc.Vault.Addr, p2Acc.Addr, ts.spec, true},
		{"stake existing provider", p1Acc.Vault.Addr, p1Acc.Addr, ts.spec, false},
		{"stake existing provider different chain", p1Acc.Vault.Addr, p1Acc.Addr, spec1, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			vaultBefore := ts.GetBalance(tt.vault)
			providerBefore := ts.GetBalance(tt.provider)
			err := ts.StakeProviderExtra(tt.vault.String(), tt.provider.String(), tt.spec, testStake, []epochstoragetypes.Endpoint{{Geolocation: 1}}, 1, "", "", "", "", "")
			vaultAfter := ts.GetBalance(tt.vault)
			providerAfter := ts.GetBalance(tt.provider)

			ts.AdvanceEpoch()

			if !tt.valid {
				require.Error(t, err)

				// balance
				require.Equal(t, vaultBefore, vaultAfter)
				require.Equal(t, providerBefore, providerAfter)

				// stake entry
				_, found := ts.Keepers.Epochstorage.GetStakeEntryByAddressCurrent(ts.Ctx, tt.spec.Index, tt.provider.String())
				require.True(t, found) // should be found because provider is registered in a vaild stake entry

				// delegations
				res, err := ts.QueryDualstakingDelegatorProviders(tt.vault.String(), false)
				require.NoError(t, err)
				require.Len(t, res.Delegations, 0)
			} else {
				require.NoError(t, err)

				// balance
				require.Equal(t, vaultBefore-testStake, vaultAfter)
				if tt.provider.Equals(tt.vault) {
					require.Equal(t, providerBefore-testStake, providerAfter)
				} else {
					require.Equal(t, providerBefore, providerAfter)
				}

				// stake entry
				stakeEntry, found := ts.Keepers.Epochstorage.GetStakeEntryByAddressCurrent(ts.Ctx, tt.spec.Index, tt.provider.String())
				require.True(t, found)
				require.Equal(t, testStake, stakeEntry.Stake.Amount.Int64())
				require.Equal(t, int64(0), stakeEntry.DelegateTotal.Amount.Int64())

				// delegations
				res, err := ts.QueryDualstakingDelegatorProviders(tt.vault.String(), false)
				require.NoError(t, err)
				require.Len(t, res.Delegations, 1)
				require.Equal(t, tt.provider.String(), res.Delegations[0].Provider)
				require.Equal(t, testStake, res.Delegations[0].Amount.Amount.Int64())
			}
		})
	}
}

// TestVaultProviderExistingStakeEntry tests that staking (or "self delegation") to an existing stake entry
// only works when the creator is the vault and not the provider.
// For each scenario, we'll check the vault's balance, the stake entry's stake/delegate total fields and the delegations
// fixation store.
// Scenarios:
// 1. stake with vault to an existing stake entry -> should work
// 2. try with provider -> should fail
func TestVaultProviderExistingStakeEntry(t *testing.T) {
	ts := newTester(t)

	p1Acc, _ := ts.AddAccount(common.PROVIDER, 0, testBalance)
	err := ts.StakeProviderExtra(p1Acc.GetVaultAddr(), p1Acc.Addr.String(), ts.spec, testStake, []epochstoragetypes.Endpoint{{Geolocation: 1}}, 1, "", "", "", "", "")
	require.NoError(t, err)

	tests := []struct {
		name     string
		vault    sdk.AccAddress
		provider sdk.AccAddress
		spec     spectypes.Spec
		valid    bool
	}{
		{"stake with vault", p1Acc.Vault.Addr, p1Acc.Addr, ts.spec, true},
		{"stake with provider", p1Acc.Addr, p1Acc.Addr, ts.spec, false},
	}

	for i, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			beforeVault := ts.GetBalance(tt.vault)
			beforeProvider := ts.GetBalance(tt.provider)
			fundsToStake := testStake + (100 * int64(i+1)) // funds to stake. needs to change each iteration for the checks below
			err := ts.StakeProviderExtra(tt.vault.String(), tt.provider.String(), tt.spec, fundsToStake, []epochstoragetypes.Endpoint{{Geolocation: 1}}, 1, "", "", "", "", "")
			afterVault := ts.GetBalance(tt.vault)
			afterProvider := ts.GetBalance(tt.provider)

			ts.AdvanceEpoch()

			if !tt.valid {
				require.Error(t, err)

				// balance
				require.Equal(t, beforeVault, afterVault)
				require.Equal(t, beforeProvider, afterProvider)

				// stake entry
				_, found := ts.Keepers.Epochstorage.GetStakeEntryByAddressCurrent(ts.Ctx, tt.spec.Index, tt.provider.String())
				require.True(t, found) // should be found because provider is registered in a vaild stake entry

				// delegations
				res, err := ts.QueryDualstakingDelegatorProviders(tt.vault.String(), false)
				require.NoError(t, err)
				require.Len(t, res.Delegations, 0)
			} else {
				require.NoError(t, err)

				// balance
				require.Equal(t, beforeVault-100, afterVault)
				if tt.provider.Equals(tt.vault) {
					require.Equal(t, beforeProvider-100, afterProvider)
				} else {
					require.Equal(t, beforeProvider, afterProvider)
				}

				// stake entry
				stakeEntry, found := ts.Keepers.Epochstorage.GetStakeEntryByAddressCurrent(ts.Ctx, tt.spec.Index, tt.provider.String())
				require.True(t, found)
				require.Equal(t, testStake+100, stakeEntry.Stake.Amount.Int64())
				require.Equal(t, int64(0), stakeEntry.DelegateTotal.Amount.Int64())

				// delegations
				res, err := ts.QueryDualstakingDelegatorProviders(tt.vault.String(), false)
				require.NoError(t, err)
				require.Len(t, res.Delegations, 1)
				require.Equal(t, tt.provider.String(), res.Delegations[0].Provider)
				require.Equal(t, testStake+100, res.Delegations[0].Amount.Amount.Int64())
			}
		})
	}
}

// TestVaultProviderModifyStakeEntry tests that the provider can change non-stake related properties of the stake entry
// and the vault address can change anything
// Scenarios:
// 1. provider modifies all non-stake related properties -> should work
// 2. provider modifies all stake related properties -> should fail
// 3. vault can change anything in the stake entry
func TestVaultProviderModifyStakeEntry(t *testing.T) {
	ts := newTester(t)
	ts.setupForPayments(1, 1, 0)
	acc, _ := ts.GetAccount(common.PROVIDER, 0)
	_, dummy := ts.GetAccount(common.CONSUMER, 0)

	provider := acc.Addr.String()
	vault := acc.GetVaultAddr()
	valAcc, _ := ts.GetAccount(common.VALIDATOR, 0)

	stakeEntry, found := ts.Keepers.Epochstorage.GetStakeEntryByAddressCurrent(ts.Ctx, ts.spec.Index, acc.Addr.String())
	require.True(t, found)

	// consts for stake entry changes
	const (
		STAKE = iota + 1
		ENDPOINTS_GEOLOCATION
		DESCRIPTION
		DELEGATE_LIMIT
		DELEGATE_COMMISSION
		PROVIDER
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
		{"description change vault", vault, DESCRIPTION, true},
		{"delegate total change vault", vault, DELEGATE_LIMIT, true},
		{"delegate commission change vault", vault, DELEGATE_COMMISSION, true},
		{"provider change vault", vault, PROVIDER, true},

		// provider can't change stake/delegation related properties
		{"stake change provider", provider, STAKE, false},
		{"endpoints_geo change provider", provider, ENDPOINTS_GEOLOCATION, true},
		{"description change provider", provider, DESCRIPTION, true},
		{"delegate total change provider", provider, DELEGATE_LIMIT, false},
		{"delegate commission change provider", provider, DELEGATE_COMMISSION, false},
		{"provider change provider", provider, PROVIDER, true},
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
				DelegateLimit:      stakeEntry.DelegateLimit,
				DelegateCommission: stakeEntry.DelegateCommission,
				Address:            stakeEntry.Address,
				Description:        stakeEntry.Description,
			}

			switch tt.stakeChange {
			case STAKE:
				msg.Amount = msg.Amount.AddAmount(math.NewInt(100))
			case ENDPOINTS_GEOLOCATION:
				msg.Endpoints = []epochstoragetypes.Endpoint{{Geolocation: 2}}
				msg.Geolocation = 2
			case DESCRIPTION:
				msg.Description = stakingtypes.NewDescription("bla", "bla", "", "", "")
			case DELEGATE_LIMIT:
				msg.DelegateLimit = msg.DelegateLimit.AddAmount(math.NewInt(100))
			case DELEGATE_COMMISSION:
				msg.DelegateCommission -= 10
			case PROVIDER:
				provider = dummy
			}

			_, err := ts.TxPairingStakeProviderFull(
				msg.Creator,
				msg.Address,
				msg.ChainID,
				msg.Amount,
				msg.Endpoints,
				msg.Geolocation,
				msg.DelegateCommission,
				msg.DelegateLimit.Amount.Uint64(),
				msg.Description.Moniker,
				msg.Description.Identity,
				msg.Description.Website,
				msg.Description.SecurityContact,
				msg.Description.Details,
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

// TestDelegatorStakesAfterProviderUnstakes tests the following scenario:
// There's a staked provider and a delegator delegates to it. Then the provider unstakes (delegation entry is still there).
// Now, the delegator stakes:
//  1. as the vault
//  2. as the provider address
//
// Both (separately) should be successful. Only the provider's vault/provider address cannot stake if they are already registered in a stake
// entry. The delegator is a third-party account, so it should not be influenced by the stake restrictions.
func TestDelegatorStakesAfterProviderUnstakes(t *testing.T) {
	tests := []struct {
		name             string
		delegatorIsVault bool // when false, delegator stakes as provider address
	}{
		{"delegator stakes as vault", true},
		{"delegator stakes as provider address", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ts := newTester(t)

			// create staked provider
			ts.setupForPayments(1, 0, 0)
			acc, provider := ts.GetAccount(common.PROVIDER, 0)
			vault := acc.Vault.Addr.String()

			// create delegator and delegate to provider
			_, delegator := ts.AddAccount(common.CONSUMER, 0, testBalance)
			_, err := ts.TxDualstakingDelegate(delegator, provider, ts.spec.Index, common.NewCoin(ts.TokenDenom(), testBalance/4))
			require.NoError(t, err)
			ts.AdvanceEpoch()

			// unstake the provider and verify the delegation still exists
			_, err = ts.TxPairingUnstakeProvider(vault, ts.spec.Index)
			require.NoError(t, err)

			res, err := ts.QueryDualstakingDelegatorProviders(delegator, false)
			require.NoError(t, err)
			require.Len(t, res.Delegations, 1)
			require.Equal(t, delegator, res.Delegations[0].Delegator)
			require.Equal(t, provider, res.Delegations[0].Provider)

			// try to stake with the delegator on the same chain as vault/operator
			if tt.delegatorIsVault {
				err = ts.StakeProvider(delegator, delegator, ts.spec, testBalance/2)
				require.NoError(t, err)
			} else {
				_, dummyVault := ts.AddAccount(common.CONSUMER, 0, testBalance)
				err = ts.StakeProvider(dummyVault, delegator, ts.spec, testBalance/2)
				require.NoError(t, err)
			}
		})
	}
}

// TestDelegatorAfterProviderUnstakeAndStake tests the following scenarios:
// Assume 3 different accounts: banana, apple and orange
// For both scenarios, banana stakes and apple delegates to it. Then banana unstakes. Now, there are two scenarios:
//   1. orange stakes with banana provider address
//   2. banana stakes with orange provider address

// When delegating, the delegator delegates to the provider address. So the delegation should be apple->banana.
// In the first case, apple should be able to unbond and banana has no delegations after it.
// In the second case, apple should not be delegated to banana (because now banana is only the vault) and apple should be able to unbond.
func TestDelegatorAfterProviderUnstakeAndStake(t *testing.T) {
	tests := []struct {
		name              string
		restakeWithOrange bool // true = first scenario, false = second scenario
	}{
		{"orange stakes with banana provider address", true},
		{"banana stakes with orange provider address", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ts := newTester(t)

			// create banana staked provider
			_, banana := ts.AddAccount(common.PROVIDER, 0, testBalance)
			err := ts.StakeProvider(banana, banana, ts.spec, testBalance/2)
			require.NoError(t, err)

			// create apple apple and delegate to banana
			_, apple := ts.AddAccount(common.CONSUMER, 0, testBalance)
			_, err = ts.TxDualstakingDelegate(apple, banana, ts.spec.Index, common.NewCoin(ts.TokenDenom(), testBalance/4))
			require.NoError(t, err)
			ts.AdvanceEpoch()

			// unstake banana and verify the delegation still exists
			_, err = ts.TxPairingUnstakeProvider(banana, ts.spec.Index)
			require.NoError(t, err)
			ts.AdvanceEpoch()

			res, err := ts.QueryDualstakingDelegatorProviders(apple, false)
			require.NoError(t, err)
			require.Len(t, res.Delegations, 1)
			require.Equal(t, apple, res.Delegations[0].Delegator)
			require.Equal(t, banana, res.Delegations[0].Provider)

			// create orange account
			_, orange := ts.AddAccount(common.CONSUMER, 1, testBalance)

			// check both scenarios
			if tt.restakeWithOrange {
				// stake with orange vault and banana provider address
				// total delegations of new stake should not be zero
				// apple should be able to unbond. After this, banana has one delegation: orange->banana
				err = ts.StakeProvider(orange, banana, ts.spec, testBalance/4)
				require.NoError(t, err)
				stakeEntry, found := ts.Keepers.Epochstorage.GetStakeEntryByAddressCurrent(ts.Ctx, ts.spec.Index, orange)
				require.True(t, found)
				require.NotZero(t, stakeEntry.DelegateTotal.Amount.Int64())

				_, err = ts.TxDualstakingUnbond(apple, banana, ts.spec.Index, common.NewCoin(ts.TokenDenom(), testBalance/4))
				require.NoError(t, err)
				ts.AdvanceEpoch()
				res, err := ts.QueryDualstakingProviderDelegators(banana, false)
				require.NoError(t, err)
				require.Len(t, res.Delegations, 1)
				require.Equal(t, banana, res.Delegations[0].Provider)
				require.Equal(t, orange, res.Delegations[0].Delegator)
			} else {
				// stake with banana vault and orange provider address
				// total delegations of new stake should be zero
				// apple should be able to unbond. After this, banana has no delegations
				err = ts.StakeProvider(banana, orange, ts.spec, testBalance/4)
				require.NoError(t, err)
				stakeEntry, found := ts.Keepers.Epochstorage.GetStakeEntryByAddressCurrent(ts.Ctx, ts.spec.Index, banana)
				require.True(t, found)
				require.Zero(t, stakeEntry.DelegateTotal.Amount.Int64())

				_, err = ts.TxDualstakingUnbond(apple, banana, ts.spec.Index, common.NewCoin(ts.TokenDenom(), testBalance/4))
				require.NoError(t, err)
				ts.AdvanceEpoch()
				res, err := ts.QueryDualstakingProviderDelegators(banana, false)
				require.NoError(t, err)
				require.Len(t, res.Delegations, 0)
			}
		})
	}
}
