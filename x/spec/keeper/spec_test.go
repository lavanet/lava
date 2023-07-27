package keeper_test

import (
	"os"
	"strconv"
	"testing"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/testutil/common"
	keepertest "github.com/lavanet/lava/testutil/keeper"
	"github.com/lavanet/lava/testutil/nullify"
	"github.com/lavanet/lava/x/spec/client/utils"
	"github.com/lavanet/lava/x/spec/keeper"
	"github.com/lavanet/lava/x/spec/types"
	"github.com/stretchr/testify/require"
)

type tester struct {
	common.Tester
}

func newTester(t *testing.T) *tester {
	return &tester{Tester: *common.NewTester(t)}
}

func (ts *tester) getSpec(index string) (types.Spec, bool) {
	return ts.Keepers.Spec.GetSpec(ts.Ctx, index)
}

func (ts *tester) setSpec(spec types.Spec) {
	ts.Keepers.Spec.SetSpec(ts.Ctx, spec)
}

func (ts *tester) removeSpec(index string) {
	ts.Keepers.Spec.RemoveSpec(ts.Ctx, index)
}

func (ts *tester) expandSpec(spec types.Spec) (types.Spec, error) {
	return ts.Keepers.Spec.ExpandSpec(ts.Ctx, spec)
}

// prepareMockApis returns a slice of mock ServiceApi for use in Spec
func prepareMockApis(count int, apiDiff string) []*types.Api {
	if count%2 != 0 {
		count += 1
	}
	mockApis := make([]*types.Api, count)

	for i := 0; i < count/2; i++ {
		api := &types.Api{
			Name: "API-" + strconv.Itoa(i),
			BlockParsing: types.BlockParser{
				DefaultValue: apiDiff,
			},
		}

		api.Enabled = true
		mockApis[i] = api

		api = &types.Api{
			Name:    "API-" + strconv.Itoa(i+count/2),
			Enabled: false,
			BlockParsing: types.BlockParser{
				DefaultValue: apiDiff,
			},
		}
		mockApis[i+count/2] = api
	}

	return mockApis
}

func prepareMockParsing(count int) []*types.ParseDirective {
	mockParsing := make([]*types.ParseDirective, count)

	for i := 0; i < count; i++ {
		uniqueName := "API-" + strconv.Itoa(i)
		api := &types.Api{
			Name: uniqueName,
		}

		api.Enabled = true
		parsing := &types.ParseDirective{
			FunctionTag:      types.FUNCTION_TAG(i + 1),
			FunctionTemplate: "%s",
			ResultParsing:    types.BlockParser{},
			ApiName:          api.Name,
		}
		mockParsing[i] = parsing
	}

	return mockParsing
}

func createApiCollection(
	apiCount int,
	apiIds []int,
	parsingCount int,
	apiInterface string,
	connectionType string,
	addon string,
	imports []*types.CollectionData,
	apiDiff string,
) *types.ApiCollection {
	return &types.ApiCollection{
		Enabled: true,
		CollectionData: types.CollectionData{
			ApiInterface: apiInterface,
			InternalPath: "",
			Type:         connectionType,
			AddOn:        addon,
		},
		Apis:            selectMockApis(prepareMockApis(apiCount, apiDiff), apiIds),
		Headers:         []*types.Header{},
		InheritanceApis: imports,
		ParseDirectives: prepareMockParsing(parsingCount),
	}
}

func generateHeaders(count int) (retHeaders []*types.Header) {
	retHeaders = []*types.Header{}
	for i := 0; i < count; i++ {
		header := &types.Header{
			Name: "header" + strconv.Itoa(i),
			Kind: 0,
		}
		retHeaders = append(retHeaders, header)
	}
	return
}

func createApiCollectionWithHeaders(
	apiCount int,
	apiIds []int,
	parsingCount int,
	headersCount int,
	apiInterface string,
	connectionType string,
	addon string,
	imports []*types.CollectionData,
	apiDiff string,
) *types.ApiCollection {
	apiCollection := createApiCollection(
		apiCount,
		apiIds,
		parsingCount,
		apiInterface,
		connectionType,
		addon,
		imports,
		apiDiff)
	apiCollection.Headers = generateHeaders(headersCount)
	return apiCollection
}

// selectMockApis returns a slice of ServiceApi corresponding to the given ids
func selectMockApis(apis []*types.Api, ids []int) []*types.Api {
	var res []*types.Api

	for _, i := range ids {
		res = append(res, apis[i])
	}

	return res
}

// createNSpec retruns a slice of mock simple Spec
func createNSpec(keeper *keeper.Keeper, ctx sdk.Context, n int) []types.Spec {
	items := make([]types.Spec, n)
	for i := range items {
		items[i].Index = strconv.Itoa(i)
		keeper.SetSpec(ctx, items[i])
	}
	return items
}

func (ts *tester) createNSpec(n int) []types.Spec {
	return createNSpec(&ts.Keepers.Spec, ts.Ctx, n)
}

func TestSpecGet(t *testing.T) {
	ts := newTester(t)
	items := ts.createNSpec(10)
	for _, item := range items {
		rst, found := ts.getSpec(item.Index)
		require.True(t, found)
		require.Equal(t,
			nullify.Fill(&item),
			nullify.Fill(&rst),
		)
	}
}

func TestSpecRemove(t *testing.T) {
	ts := newTester(t)
	items := ts.createNSpec(10)
	for _, item := range items {
		ts.removeSpec(item.Index)
		_, found := ts.getSpec(item.Index)
		require.False(t, found)
	}
}

func TestSpecGetAll(t *testing.T) {
	ts := newTester(t)
	items := ts.createNSpec(10)
	rst := ts.Keepers.Spec.GetAllSpec(ts.Ctx)
	require.ElementsMatch(t,
		nullify.Fill(items),
		nullify.Fill(rst),
	)
}

// setupSpecsForSpecInheritance returns a slice of Spec according to the
// template therein, to simulate collection of existing Spec(s) on the chain.
func (ts *tester) setupSpecsForSpecInheritance(apis []*types.Api, apisDiff []*types.Api) {
	template := []struct {
		name    string
		enabled bool
		imports []string
		apis    []int
		apiDiff bool
	}{
		{name: "disabled", enabled: false, imports: nil, apis: []int{0, 2}},
		{name: "one-two", enabled: true, imports: nil, apis: []int{0, 2}},
		{name: "oneX-three", enabled: true, imports: nil, apis: []int{1, 4}},
		{name: "three-four", enabled: true, imports: nil, apis: []int{1, 3}},
		{name: "threeX-four", enabled: true, imports: nil, apis: []int{3, 6}, apiDiff: true},
	}

	for _, tt := range template {
		apisToSpec := selectMockApis(apis, tt.apis)
		if tt.apiDiff {
			apisToSpec = selectMockApis(apisDiff, tt.apis)
		}
		apiCollection := &types.ApiCollection{
			Enabled:        true,
			CollectionData: types.CollectionData{ApiInterface: "stub"},
			Apis:           apisToSpec,
		}
		spec := types.Spec{
			Name:           tt.name,
			Index:          tt.name,
			Enabled:        tt.enabled,
			ApiCollections: []*types.ApiCollection{apiCollection},
		}
		ts.AddSpec(tt.name, spec) // also calls SetSpec()
	}
}

// Note: the API identifiers below refer to the APIs from the function
// setupSpecsForSpecInheritance() above
var specTemplates = []struct {
	desc    string
	name    string
	imports []string
	apis    []int
	result  []int
	ok      bool
}{
	{
		name:    "import:unknown",
		desc:    "import from unknown spec [expected: ERROR]",
		imports: []string{"non-existent"},
		apis:    nil,
		result:  nil,
		ok:      false,
	},
	{
		name:    "import:disabled",
		desc:    "import from disabled spec",
		imports: []string{"disabled"},
		apis:    nil,
		result:  []int{0, 2},
		ok:      true,
	},
	{
		name:    "import:one-spec",
		desc:    "import one spec",
		imports: []string{"one-two"},
		apis:    nil,
		result:  []int{0, 2},
		ok:      true,
	},
	{
		name:    "import:with-override",
		desc:    "import one spec with override in current spec",
		imports: []string{"one-two"},
		apis:    []int{1},
		result:  []int{0, 1, 2},
		ok:      true,
	},
	{
		name:    "import:two-spec",
		desc:    "import two specs",
		imports: []string{"one-two", "three-four"},
		apis:    nil,
		result:  []int{0, 1, 2, 3},
		ok:      true,
	},
	{
		name:    "import:duplicate",
		desc:    "import two specs with duplicate api [expected: ERROR]",
		imports: []string{"three-four", "threeX-four"},
		apis:    nil,
		result:  nil,
		ok:      false,
	},
	{
		name:    "import:with-override-dup",
		desc:    "import two specs with duplicate api with override in current spec",
		imports: []string{"three-four", "threeX-four"},
		apis:    []int{3},
		result:  []int{1, 3},
		ok:      true,
	},
	{
		name:    "import:two-level",
		desc:    "import two level (one spec that imports another)",
		imports: []string{"import:one-spec"}, // assumes 'import:one-spec' already added
		apis:    []int{4},
		result:  []int{4, 0, 2},
		ok:      true,
	},
	{
		name:    "import:two-level",
		desc:    "import two level (one spec that imports another) with disabled",
		imports: []string{"import:with-override"}, // assumes 'import:with-override' already added
		apis:    nil,
		result:  []int{0, 1, 2},
		ok:      true,
	},
	{
		desc:    "import self spec [expected: ERROR]",
		name:    "import:self",
		imports: []string{"import:self"},
		apis:    nil,
		result:  nil,
		ok:      false,
	},
	{
		desc:    "import with loop (modify 'one-two' imported by 'import:one-spec')",
		name:    "one-two",
		imports: []string{"import:one-spec"}, // assumes 'import:one-spec' already added
		apis:    nil,
		result:  nil,
		ok:      false,
	},
}

// Test Spec with "import" directives (using to the templates above).
func TestSpecWithImport(t *testing.T) {
	ts := newTester(t)

	apis := prepareMockApis(8, "")
	apisDiff := prepareMockApis(8, "X")
	ts.setupSpecsForSpecInheritance(apis, apisDiff)

	for _, tt := range specTemplates {
		sp := types.Spec{
			Name:    tt.name,
			Index:   tt.name,
			Imports: tt.imports,
			Enabled: true,
			ApiCollections: []*types.ApiCollection{
				{
					Enabled: true,
					CollectionData: types.CollectionData{
						ApiInterface: "stub",
					},
					Apis: selectMockApis(apis, tt.apis),
				},
			},
		}

		t.Run(tt.desc, func(t *testing.T) {
			fullspec, err := ts.expandSpec(sp)
			if tt.ok == true {
				require.Nil(t, err, err)
				require.Len(t, fullspec.ApiCollections[0].Apis, len(tt.result))

				for i := 0; i < len(tt.result); i++ {
					nameToFind := apis[tt.result[i]].Name
					found := false
					for _, api := range fullspec.ApiCollections[0].Apis {
						if api.Name == nameToFind {
							require.False(t, found) // only found once
							found = true
							require.Equal(t, api.Enabled, apis[tt.result[i]].Enabled)
						}
					}
					require.True(t, found)
				}

				ts.setSpec(sp)
			} else {
				require.NotNil(t, err)
			}
		})
	}
}

func (ts *tester) setupSpecsForApiInheritance() {
	template := []struct {
		name           string
		enabled        bool
		imports        []string
		apiCollections []*types.ApiCollection
	}{
		{name: "disabled", enabled: false, imports: nil, apiCollections: []*types.ApiCollection{
			createApiCollectionWithHeaders(20, []int{0, 4, 10}, 1, 2, "test1", "", "", nil, ""),
		}},
		{name: "one", enabled: true, imports: nil, apiCollections: []*types.ApiCollection{
			createApiCollectionWithHeaders(20, []int{0, 4, 11}, 1, 2, "test1", "", "", nil, ""),
			createApiCollectionWithHeaders(20, []int{0, 4, 11}, 1, 2, "test-one", "", "", nil, ""),
		}},
		{name: "two", enabled: true, imports: nil, apiCollections: []*types.ApiCollection{
			createApiCollectionWithHeaders(20, []int{1, 5, 12}, 1, 2, "test1", "", "", nil, ""),
			createApiCollectionWithHeaders(20, []int{1, 5, 12}, 1, 2, "test1", "test", "", nil, ""),
			createApiCollectionWithHeaders(20, []int{0, 4, 12}, 1, 2, "test-two", "", "", nil, ""),
		}},
		{name: "three", enabled: true, imports: nil, apiCollections: []*types.ApiCollection{
			createApiCollectionWithHeaders(20, []int{2, 6, 13}, 1, 2, "test1", "", "", nil, ""),
			createApiCollectionWithHeaders(20, []int{2, 6, 13}, 1, 2, "test1", "", "test1", nil, ""),
			createApiCollectionWithHeaders(20, []int{0}, 0, 0, "test1", "", "test2", []*types.CollectionData{&createApiCollection(20, []int{2, 6}, 1, "test1", "", "", nil, "").CollectionData}, ""),
			createApiCollectionWithHeaders(20, []int{0, 4, 13}, 1, 2, "test-three", "", "", nil, ""),
		}},
		{name: "one-conflict", enabled: true, imports: nil, apiCollections: []*types.ApiCollection{
			createApiCollectionWithHeaders(20, []int{0, 3, 11}, 1, 2, "test1", "", "", nil, "diff"),
		}},
		{name: "two-conflict", enabled: true, imports: nil, apiCollections: []*types.ApiCollection{
			createApiCollectionWithHeaders(20, []int{1, 3, 12}, 1, 2, "test1", "", "", nil, "diff"),
			createApiCollectionWithHeaders(20, []int{1, 3, 12}, 1, 2, "test1", "test", "", nil, "diff"),
		}},
		{name: "three-conflict", enabled: true, imports: nil, apiCollections: []*types.ApiCollection{
			createApiCollectionWithHeaders(20, []int{2, 3, 13}, 1, 2, "test1", "", "", nil, "diff"),
			createApiCollectionWithHeaders(20, []int{2, 3, 13}, 1, 2, "test1", "", "test1", nil, "diff"),
			createApiCollectionWithHeaders(20, []int{0, 3, 13}, 0, 0, "test1", "", "test2", []*types.CollectionData{&createApiCollection(20, []int{2, 6}, 1, "test1", "", "", nil, "diff").CollectionData}, "diff"),
		}},
	}

	for _, tt := range template {
		spec := types.Spec{
			Name:           tt.name,
			Index:          tt.name,
			Enabled:        tt.enabled,
			ApiCollections: tt.apiCollections,
		}
		ts.AddSpec(tt.name, spec) // also calls SetSpec()
	}
}

func TestSpecUpdateInherit(t *testing.T) {
	ts := newTester(t)

	api := types.Api{
		Enabled:      true,
		Name:         "eth_blockNumber",
		ComputeUnits: 10,
	}

	parentSpec := types.Spec{
		Index:                         "parent",
		Name:                          "parent spec",
		Enabled:                       true,
		ReliabilityThreshold:          268435455,
		DataReliabilityEnabled:        false,
		BlockDistanceForFinalizedData: 64,
		BlocksInFinalizationProof:     3,
		AverageBlockTime:              13000,
		AllowedBlockLagForQosSync:     2,
		MinStakeProvider:              common.NewCoin(5000),
		MinStakeClient:                common.NewCoin(5000),
		ApiCollections: []*types.ApiCollection{
			{
				Enabled:        true,
				CollectionData: types.CollectionData{ApiInterface: "jsonrpc"},
				Apis:           []*types.Api{&api},
			},
		},
	}

	childSpec := types.Spec{
		Index:                         "child",
		Name:                          "child spec",
		Enabled:                       true,
		ReliabilityThreshold:          268435455,
		DataReliabilityEnabled:        false,
		BlockDistanceForFinalizedData: 64,
		BlocksInFinalizationProof:     3,
		AverageBlockTime:              13000,
		AllowedBlockLagForQosSync:     2,
		MinStakeProvider:              common.NewCoin(5000),
		MinStakeClient:                common.NewCoin(5000),
		Imports:                       []string{"parent"},
	}

	// add a parent spec and a child spec
	err := keepertest.SimulateSpecAddProposal(ts.Ctx, ts.Keepers.Spec, []types.Spec{parentSpec})
	require.Nil(t, err)

	err = keepertest.SimulateSpecAddProposal(ts.Ctx, ts.Keepers.Spec, []types.Spec{childSpec})
	require.Nil(t, err)

	block1 := ts.BlockHeight()

	sp, found := ts.getSpec("child")
	require.True(t, found)
	sp, err = ts.expandSpec(sp)
	require.Nil(t, err)
	require.Equal(t, uint64(10), sp.ApiCollections[0].Apis[0].ComputeUnits)
	require.Equal(t, block1, sp.BlockLastUpdated)

	ts.AdvanceBlock()

	// modify the parent spec and verify that the child is refreshed

	parentSpec.ApiCollections[0].Apis[0].ComputeUnits = 20
	err = keepertest.SimulateSpecAddProposal(ts.Ctx, ts.Keepers.Spec, []types.Spec{parentSpec})
	require.Nil(t, err)

	sp, found = ts.getSpec("parent")
	require.True(t, found)
	require.Equal(t, uint64(20), sp.ApiCollections[0].Apis[0].ComputeUnits)
	require.Equal(t, block1+1, sp.BlockLastUpdated)

	sp, found = ts.getSpec("child")
	require.True(t, found)
	sp, err = ts.expandSpec(sp)
	require.Nil(t, err)
	require.Equal(t, uint64(20), sp.ApiCollections[0].Apis[0].ComputeUnits)
	require.Equal(t, block1+1, sp.BlockLastUpdated)
}

func TestApiCollectionsExpandAndInheritance(t *testing.T) {
	ts := newTester(t)

	apis := prepareMockApis(20, "")
	ts.setupSpecsForApiInheritance()

	specTemplates := []struct {
		desc                 string
		name                 string
		imports              []string
		apisCollections      []*types.ApiCollection
		resultApiCollections int
		result               []int // the apis inside api collection: "test1", "", "",
		totalApis            int   // total enabled apis in all api collections
		ok                   bool
	}{
		{
			name:    "expand",
			desc:    "with several api collections expanding from each other",
			imports: nil,
			apisCollections: []*types.ApiCollection{
				createApiCollectionWithHeaders(20, []int{0, 1}, 1, 2, "", "", "", nil, ""),
				createApiCollectionWithHeaders(20, []int{1, 2}, 0, 0, "test1", "", "", []*types.CollectionData{&createApiCollection(20, []int{0, 1}, 1, "", "", "", nil, "").CollectionData}, ""),
				createApiCollectionWithHeaders(0, []int{}, 0, 0, "test1", "", "addon", []*types.CollectionData{&createApiCollection(20, []int{0, 1}, 1, "test1", "", "", nil, "").CollectionData}, ""),
			},
			result:               []int{0, 1, 2},
			resultApiCollections: 3,
			totalApis:            8,
			ok:                   true,
		},
		{
			name:    "expand-fail",
			desc:    "fail on several api collections expanding from each other",
			imports: nil,
			apisCollections: []*types.ApiCollection{
				createApiCollectionWithHeaders(20, []int{0, 1}, 1, 2, "123", "", "", nil, ""),
				createApiCollectionWithHeaders(20, []int{1, 2}, 1, 2, "test1", "", "", []*types.CollectionData{&createApiCollection(20, []int{0, 1}, 1, "non-existent", "", "", nil, "").CollectionData}, ""),
			},
			ok: false,
		},
		{
			name:    "expand-fail2",
			desc:    "fail on expand of a incompatible apiInterface type",
			imports: nil,
			apisCollections: []*types.ApiCollection{
				createApiCollectionWithHeaders(20, []int{0, 1}, 1, 2, "test1", "", "", nil, ""),
				createApiCollectionWithHeaders(20, []int{1, 2}, 1, 2, "test2", "", "", []*types.CollectionData{&createApiCollection(20, []int{0, 1}, 1, "test1", "", "", nil, "").CollectionData}, ""),
			},
			ok: false,
		},
		{
			name:            "import:unknown",
			desc:            "import from unknown spec [expected: ERROR]",
			imports:         []string{"non-existent"},
			apisCollections: nil,
			ok:              false,
		},
		{
			name:                 "import:disabled",
			desc:                 "import from disabled spec",
			imports:              []string{"disabled"},
			apisCollections:      nil,
			result:               []int{0, 4},
			resultApiCollections: 1,
			totalApis:            2,
			ok:                   true,
		},
		{
			name:                 "import:one-spec",
			desc:                 "import one spec",
			imports:              []string{"one"},
			apisCollections:      nil,
			result:               []int{0, 4},
			resultApiCollections: 2,
			totalApis:            4,
			ok:                   true,
		},
		{
			name:                 "import:spec-two",
			desc:                 "import one spec called two",
			imports:              []string{"two"},
			apisCollections:      nil,
			result:               []int{1, 5},
			resultApiCollections: 3,
			totalApis:            6,
			ok:                   true,
		},
		{
			name:                 "import:spec-three",
			desc:                 "import one spec called three",
			imports:              []string{"three"},
			apisCollections:      nil,
			result:               []int{2, 6},
			resultApiCollections: 4,
			totalApis:            9,
			ok:                   true,
		},
		{
			name:                 "import:with-override",
			desc:                 "import one spec with override in current spec",
			imports:              []string{"one"},
			apisCollections:      []*types.ApiCollection{createApiCollectionWithHeaders(20, []int{0, 1}, 1, 2, "test1", "", "", nil, "")},
			result:               []int{0, 1, 4},
			resultApiCollections: 2,
			totalApis:            5,
			ok:                   true,
		},
		{
			name:                 "import:with-no-overlap",
			desc:                 "import one spec with no overlap in collections in current spec",
			imports:              []string{"one"},
			apisCollections:      []*types.ApiCollection{createApiCollectionWithHeaders(20, []int{0, 1}, 1, 2, "test-no-overlap", "", "", nil, "")},
			result:               []int{0, 4},
			resultApiCollections: 3,
			totalApis:            6,
			ok:                   true,
		},
		{
			name:                 "import:two-specs",
			desc:                 "import two specs",
			imports:              []string{"one", "two"},
			apisCollections:      nil,
			result:               []int{0, 1, 4, 5},
			resultApiCollections: 4,
			totalApis:            10,
			ok:                   true,
		},
		{
			name:    "import:duplicate",
			desc:    "import two specs with duplicate api [expected: ERROR]",
			imports: []string{"one", "one-conflict"},
			result:  nil,
			ok:      false,
		},
		{
			name:                 "import:with-override-dup",
			desc:                 "import two specs with duplicate api with override in current spec",
			imports:              []string{"one", "one-conflict"},
			apisCollections:      []*types.ApiCollection{createApiCollectionWithHeaders(20, []int{0, 1}, 1, 2, "test1", "", "", nil, "")},
			result:               []int{0, 1, 3, 4},
			resultApiCollections: 2,
			totalApis:            6,
			ok:                   true,
		},
		{
			name:                 "import:two-level",
			desc:                 "import two level (one spec that imports another)",
			imports:              []string{"import:one-spec"}, // assumes 'import:one-spec' already added
			apisCollections:      nil,
			result:               []int{0, 4},
			resultApiCollections: 2,
			totalApis:            4,
			ok:                   true,
		},
		{
			name:                 "import:two-level-override",
			desc:                 "import two level (one spec that imports another) with disabled",
			imports:              []string{"import:with-override"}, // assumes 'import:with-override' already added
			apisCollections:      []*types.ApiCollection{createApiCollectionWithHeaders(20, []int{0, 1, 8}, 1, 2, "test1", "", "", nil, "")},
			result:               []int{0, 1, 4, 8},
			resultApiCollections: 2,
			totalApis:            6,
			ok:                   true,
		},
		{
			desc:    "import self spec [expected: ERROR]",
			name:    "import:self",
			imports: []string{"import:self"},
			result:  nil,
			ok:      false,
		},
		{
			desc:    "import with loop (modify 'one-two' imported by 'import:one-spec')",
			name:    "one",
			imports: []string{"import:one-spec"}, // assumes 'import:one-spec' already added
			result:  nil,
			ok:      false,
		},
	}

	for _, tt := range specTemplates {
		sp := types.Spec{
			Name:           tt.name,
			Index:          tt.name,
			Imports:        tt.imports,
			Enabled:        true,
			ApiCollections: tt.apisCollections,
		}

		t.Run(tt.desc, func(t *testing.T) {
			fullspec, err := ts.expandSpec(sp)
			if tt.ok == true {
				// check Result against the baseline spec  "test1", "", "",
				// count apis to totalApis
				// count ApiCollections
				collections := 0
				totApis := 0
				var compareCollection *types.ApiCollection
				for _, apiCol := range fullspec.ApiCollections {
					collections++
					for _, api := range apiCol.Apis {
						if api.Enabled {
							totApis += 1
						}
					}

					if (apiCol.CollectionData == types.CollectionData{
						ApiInterface: "test1",
						InternalPath: "",
						Type:         "",
						AddOn:        "",
					}) {
						compareCollection = apiCol
					}
					require.Equal(t, 1, len(apiCol.ParseDirectives), "collectionData %v, parsing %v", apiCol.CollectionData, apiCol.ParseDirectives)
					require.Equal(t, 2, len(apiCol.Headers))
				}
				require.Equal(t, tt.resultApiCollections, collections)
				require.Equal(t, tt.totalApis, totApis, fullspec)
				require.NotNil(t, compareCollection)
				require.Nil(t, err, err)
				enabledApis := 0
				for _, api := range compareCollection.Apis {
					if api.Enabled {
						enabledApis++
					}
				}
				require.Equal(t, enabledApis, len(tt.result))
				for i := 0; i < len(tt.result); i++ {
					nameToFind := apis[tt.result[i]].Name
					found := false
					for _, api := range compareCollection.Apis {
						if api.Name == nameToFind {
							require.False(t, found) // only found once
							found = true
							require.Equal(t, api.Enabled, apis[tt.result[i]].Enabled)
						}
					}
					require.True(t, found)
				}
				ts.setSpec(sp)
			} else {
				require.Error(t, err, "spec with no error although expected %s", sp.Index)
			}
		})
	}
}

func TestCookbookSpecs(t *testing.T) {
	ts := newTester(t)

	getToTopMostPath := "../../.././cookbook/specs/"
	// base specs needs to be proposed first
	baseSpecs := []string{"spec_add_ibc.json", "spec_add_cosmoswasm.json", "spec_add_cosmossdk.json", "spec_add_cosmossdk_full.json", "spec_add_ethereum.json"}

	Specs, err := getAllFilesInDirectory(getToTopMostPath)
	require.Nil(t, err)

	// remove the base specs so there wont be a duplicate
	Specs = removeSetFromSet(baseSpecs, Specs)
	Specs = append(baseSpecs, Specs...)
	for _, fileName := range Specs {
		proposal := utils.SpecAddProposalJSON{}

		contents, err := os.ReadFile(getToTopMostPath + fileName)
		require.Nil(t, err)

		err = codec.NewLegacyAmino().UnmarshalJSON(contents, &proposal)
		require.Nil(t, err)

		for _, sp := range proposal.Proposal.Specs {
			ts.setSpec(sp)
			fullspec, err := ts.expandSpec(sp)
			require.NoError(t, err)
			require.NotNil(t, fullspec)
		}
	}
}

func getAllFilesInDirectory(directory string) ([]string, error) {
	var files []string

	dir, err := os.ReadDir(directory)
	if err != nil {
		return nil, err
	}

	for _, entry := range dir {
		if entry.IsDir() {
			// Skip directories; we only want files
			continue
		}
		files = append(files, entry.Name())
	}

	return files, nil
}

func removeSetFromSet(set1, set2 []string) []string {
	// Create a map to store the elements of the first set
	elements := make(map[string]bool)
	for _, str := range set1 {
		elements[str] = true
	}

	// Create a new slice with elements of the second set that are not present in the first set
	var resultSet []string
	for _, str := range set2 {
		if !elements[str] {
			resultSet = append(resultSet, str)
		}
	}

	return resultSet
}
