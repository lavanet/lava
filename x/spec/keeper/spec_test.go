package keeper_test

import (
	"strconv"
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	keepertest "github.com/lavanet/lava/testutil/keeper"
	"github.com/lavanet/lava/testutil/nullify"
	"github.com/lavanet/lava/x/spec/keeper"
	"github.com/lavanet/lava/x/spec/types"
	"github.com/stretchr/testify/require"
)

// prepareMockApis returns a slice of mock ServiceApi for use in Spec
func prepareMockApis(count int) []*types.Api {
	if count%2 != 0 {
		count += 1
	}
	mockApis := make([]*types.Api, count)

	for i := 0; i < count/2; i++ {
		api := &types.Api{
			Name: "API-" + strconv.Itoa(i),
		}

		api.Enabled = true
		mockApis[i] = api

		api = &types.Api{
			Name:    "API-" + strconv.Itoa(i),
			Enabled: false,
		}
		mockApis[i+count/2] = api
	}

	return mockApis
}

func prepareMockParsing(count int) []*types.Parsing {
	mockParsing := make([]*types.Parsing, count)

	for i := 0; i < count; i++ {
		uniqueName := "API-" + strconv.Itoa(i)
		api := &types.Api{
			Name: uniqueName,
		}

		api.Enabled = true
		parsing := &types.Parsing{
			FunctionTag:      uniqueName,
			FunctionTemplate: "%s",
			ResultParsing:    types.BlockParser{},
			Api:              api,
		}
		mockParsing[i] = parsing
	}

	return mockParsing
}

func createApiCollection(apiCount int, apiIds []int, parsingCount int, apiInterface string, connectionType string, addon string, imports []*types.CollectionData) *types.ApiCollection {
	return &types.ApiCollection{
		Enabled: true,
		CollectionData: types.CollectionData{
			ApiInterface: apiInterface,
			InternalPath: "",
			Type:         connectionType,
			AddOn:        addon,
		},
		Apis:            selectMockApis(prepareMockApis(apiCount), apiIds),
		Headers:         []*types.Header{},
		InheritanceApis: imports,
		Parsing:         prepareMockParsing(parsingCount),
	}
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

func TestSpecGet(t *testing.T) {
	keeper, ctx := keepertest.SpecKeeper(t)
	items := createNSpec(keeper, ctx, 10)
	for _, item := range items {
		rst, found := keeper.GetSpec(ctx,
			item.Index,
		)
		require.True(t, found)
		require.Equal(t,
			nullify.Fill(&item),
			nullify.Fill(&rst),
		)
	}
}

func TestSpecRemove(t *testing.T) {
	keeper, ctx := keepertest.SpecKeeper(t)
	items := createNSpec(keeper, ctx, 10)
	for _, item := range items {
		keeper.RemoveSpec(ctx,
			item.Index,
		)
		_, found := keeper.GetSpec(ctx,
			item.Index,
		)
		require.False(t, found)
	}
}

func TestSpecGetAll(t *testing.T) {
	keeper, ctx := keepertest.SpecKeeper(t)
	items := createNSpec(keeper, ctx, 10)
	require.ElementsMatch(t,
		nullify.Fill(items),
		nullify.Fill(keeper.GetAllSpec(ctx)),
	)
}

// prepareMockCurrentSpecs returns a slice of Spec according to the template
// therein, to simulate the collection of existing Spec(s) on the chain.
func prepareMockCurrentSpecs(keeper *keeper.Keeper, ctx sdk.Context, apis []*types.Api) map[string]types.Spec {
	currentSpecs := make(map[string]types.Spec)

	template := []struct {
		name    string
		enabled bool
		imports []string
		apis    []int
	}{
		{name: "disabled", enabled: false, imports: nil, apis: []int{0, 2}},
		{name: "one-two", enabled: true, imports: nil, apis: []int{0, 2}},
		{name: "oneX-three", enabled: true, imports: nil, apis: []int{1, 4}},
		{name: "three-four", enabled: true, imports: nil, apis: []int{4, 6}},
		{name: "threeX-four", enabled: true, imports: nil, apis: []int{5, 6}},
	}

	for _, tt := range template {
		spec := types.Spec{
			Name:           tt.name,
			Index:          tt.name,
			Enabled:        tt.enabled,
			ApiCollections: []*types.ApiCollection{{CollectionData: types.CollectionData{ApiInterface: "stub"}, Apis: selectMockApis(apis, tt.apis)}},
		}
		currentSpecs[tt.name] = spec
		keeper.SetSpec(ctx, spec)
	}

	return currentSpecs
}

// Note: the API identifiers below refer to the APIs from the
// function prepareMockCurrentApis() above
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
		result:  []int{1, 2},
		ok:      true,
	},
	{
		name:    "import:two-spec",
		desc:    "import two specs",
		imports: []string{"one-two", "three-four"},
		apis:    nil,
		result:  []int{0, 2, 4, 6},
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
		apis:    []int{6},
		result:  []int{6, 4},
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
		result:  []int{2},
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
	keeper, ctx := keepertest.SpecKeeper(t)

	apis := prepareMockApis(8)
	_ = prepareMockCurrentSpecs(keeper, ctx, apis)

	for _, tt := range specTemplates {
		spec := types.Spec{
			Name:           tt.name,
			Index:          tt.name,
			Imports:        tt.imports,
			Enabled:        true,
			ApiCollections: []*types.ApiCollection{{CollectionData: types.CollectionData{ApiInterface: "stub"}, Apis: selectMockApis(apis, tt.apis)}},
		}

		t.Run(tt.desc, func(t *testing.T) {
			fullspec, err := keeper.ExpandSpec(ctx, spec)
			if tt.ok == true {
				require.Nil(t, err, err)
				require.Len(t, fullspec.ApiCollections[0].Apis, len(tt.result))
				for i := 0; i < len(tt.result); i++ {
					require.Equal(t, fullspec.ApiCollections[0].Apis[i].Name, apis[tt.result[i]].Name)
					require.Equal(t, fullspec.ApiCollections[0].Apis[i].Enabled, apis[tt.result[i]].Enabled)
				}
				keeper.SetSpec(ctx, fullspec)
			} else {
				require.NotNil(t, err)
			}
		})
	}
}

func prepareMockCurrentSpecsForApiCollectionInheritance(keeper *keeper.Keeper, ctx sdk.Context) map[string]types.Spec {
	currentSpecs := make(map[string]types.Spec)

	template := []struct {
		name           string
		enabled        bool
		imports        []string
		apiCollections []*types.ApiCollection
	}{
		{name: "disabled", enabled: false, imports: nil, apiCollections: []*types.ApiCollection{
			createApiCollection(20, []int{0, 4, 10}, 1, "test1", "", "", nil),
		}},
		{name: "one", enabled: true, imports: nil, apiCollections: []*types.ApiCollection{
			createApiCollection(20, []int{0, 4, 11}, 1, "test1", "", "", nil),
			createApiCollection(20, []int{0, 4, 11}, 1, "test-one", "", "", nil),
		}},
		{name: "two", enabled: true, imports: nil, apiCollections: []*types.ApiCollection{
			createApiCollection(20, []int{1, 5, 12}, 1, "test1", "", "", nil),
			createApiCollection(20, []int{1, 5, 12}, 1, "test1", "test", "", nil),
			createApiCollection(20, []int{0, 4, 12}, 1, "test-two", "", "", nil),
		}},
		{name: "three", enabled: true, imports: nil, apiCollections: []*types.ApiCollection{
			createApiCollection(20, []int{2, 6, 13}, 1, "test1", "", "", nil),
			createApiCollection(20, []int{2, 6, 13}, 1, "test1", "", "test1", nil),
			createApiCollection(20, []int{0}, 0, "test1", "", "test2", []*types.CollectionData{&createApiCollection(20, []int{2, 6}, 1, "test1", "", "", nil).CollectionData}),
			createApiCollection(20, []int{0, 4, 13}, 1, "test-three", "", "", nil),
		}},
		{name: "one-conflict", enabled: true, imports: nil, apiCollections: []*types.ApiCollection{
			createApiCollection(20, []int{0, 3, 11}, 1, "test1", "", "", nil),
		}},
		{name: "two-conflict", enabled: true, imports: nil, apiCollections: []*types.ApiCollection{
			createApiCollection(20, []int{1, 3, 12}, 1, "test1", "", "", nil),
			createApiCollection(20, []int{1, 3, 12}, 1, "test1", "test", "", nil),
		}},
		{name: "three-conflict", enabled: true, imports: nil, apiCollections: []*types.ApiCollection{
			createApiCollection(20, []int{2, 3, 13}, 1, "test1", "", "", nil),
			createApiCollection(20, []int{2, 3, 13}, 1, "test1", "", "test1", nil),
			createApiCollection(20, []int{0, 3, 13}, 0, "test1", "", "test2", []*types.CollectionData{&createApiCollection(20, []int{2, 6}, 1, "test1", "", "", nil).CollectionData}),
		}},
	}

	for _, tt := range template {
		spec := types.Spec{
			Name:           tt.name,
			Index:          tt.name,
			Enabled:        tt.enabled,
			ApiCollections: tt.apiCollections,
		}
		currentSpecs[tt.name] = spec
		keeper.SetSpec(ctx, spec)
	}

	return currentSpecs
}

func TestApiCollectionsExpandAndInheritance(t *testing.T) {
	keeper, ctx := keepertest.SpecKeeper(t)
	_ = prepareMockCurrentSpecsForApiCollectionInheritance(keeper, ctx)
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
				createApiCollection(20, []int{0, 1}, 1, "", "", "", nil),
				createApiCollection(20, []int{1, 2}, 0, "test1", "", "", []*types.CollectionData{&createApiCollection(20, []int{0, 1}, 1, "", "", "", nil).CollectionData}),
				createApiCollection(0, []int{}, 0, "test1", "", "addon", []*types.CollectionData{&createApiCollection(20, []int{0, 1}, 1, "test1", "", "", nil).CollectionData}),
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
				createApiCollection(20, []int{0, 1}, 1, "123", "", "", nil),
				createApiCollection(20, []int{1, 2}, 1, "test1", "", "", []*types.CollectionData{&createApiCollection(20, []int{0, 1}, 1, "non-existent", "", "", nil).CollectionData}),
			},
			ok: false,
		},
		{
			name:    "expand-fail2",
			desc:    "fail on expand of a incompatible apiInterface type",
			imports: nil,
			apisCollections: []*types.ApiCollection{
				createApiCollection(20, []int{0, 1}, 1, "test1", "", "", nil),
				createApiCollection(20, []int{1, 2}, 1, "test2", "", "", []*types.CollectionData{&createApiCollection(20, []int{0, 1}, 1, "test1", "", "", nil).CollectionData}),
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
			apisCollections:      []*types.ApiCollection{createApiCollection(20, []int{0, 1}, 1, "test1", "", "", nil)},
			result:               []int{0, 1, 4},
			resultApiCollections: 2,
			totalApis:            5,
			ok:                   true,
		},
		{
			name:                 "import:with-no-overlap",
			desc:                 "import one spec with no overlap in collections in current spec",
			imports:              []string{"one"},
			apisCollections:      []*types.ApiCollection{createApiCollection(20, []int{0, 1}, 1, "test-no-overlap", "", "", nil)},
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
			apisCollections:      []*types.ApiCollection{createApiCollection(20, []int{0, 1}, 1, "test1", "", "", nil)},
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
			apisCollections:      []*types.ApiCollection{createApiCollection(20, []int{0, 1, 8}, 1, "test1", "", "", nil)},
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
			name:    "one-two",
			imports: []string{"import:one-spec"}, // assumes 'import:one-spec' already added
			result:  nil,
			ok:      false,
		},
	}
	apis := prepareMockApis(20)
	for _, tt := range specTemplates {
		spec := types.Spec{
			Name:           tt.name,
			Index:          tt.name,
			Imports:        tt.imports,
			Enabled:        true,
			ApiCollections: tt.apisCollections,
		}

		t.Run(tt.desc, func(t *testing.T) {
			fullspec, err := keeper.ExpandSpec(ctx, spec)
			if tt.ok == true {
				// check Result against the baseline spec  "test1", "", "",
				// count apis to totalApis
				// count ApiCollections
				collections := 0
				totApis := 0
				var compareCollection *types.ApiCollection
				for _, apiCol := range fullspec.ApiCollections {
					collections++
					totApis += len(apiCol.Apis)
					if (apiCol.CollectionData == types.CollectionData{
						ApiInterface: "test1",
						InternalPath: "",
						Type:         "",
						AddOn:        "",
					}) {
						compareCollection = apiCol
					}
					require.Equal(t, 1, len(apiCol.Parsing))
				}
				require.Equal(t, tt.resultApiCollections, collections)
				require.Equal(t, tt.totalApis, totApis, fullspec)
				require.NotNil(t, compareCollection)
				require.Nil(t, err, err)
				require.Len(t, compareCollection.Apis, len(tt.result))
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
				keeper.SetSpec(ctx, fullspec)
			} else {
				require.NotNil(t, err, err)
			}
		})
	}
}
