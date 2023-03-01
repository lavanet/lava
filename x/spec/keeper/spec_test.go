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
func prepareMockApis() []types.ServiceApi {
	mockApis := make([]types.ServiceApi, 8)

	for i := 0; i < 4; i++ {
		api := types.ServiceApi{
			Name: "API-" + strconv.Itoa(i),
		}

		api.Enabled = true
		mockApis[2*i] = api

		api.Enabled = false
		mockApis[2*i+1] = api
	}

	return mockApis
}

// selectMockApis returns a slice of ServiceApi corresponding to the given ids
func selectMockApis(apis []types.ServiceApi, ids []int) []types.ServiceApi {
	var res []types.ServiceApi

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
func prepareMockCurrentSpecs(keeper *keeper.Keeper, ctx sdk.Context, apis []types.ServiceApi) map[string]types.Spec {
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
			Name:    tt.name,
			Index:   tt.name,
			Enabled: tt.enabled,
			Apis:    selectMockApis(apis, tt.apis),
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

	apis := prepareMockApis()
	_ = prepareMockCurrentSpecs(keeper, ctx, apis)

	for _, tt := range specTemplates {
		spec := types.Spec{
			Name:    tt.name,
			Index:   tt.name,
			Imports: tt.imports,
			Enabled: true,
			Apis:    selectMockApis(apis, tt.apis),
		}

		t.Run(tt.desc, func(t *testing.T) {
			fullspec, err := keeper.ExpandSpec(ctx, spec)
			if tt.ok == true {
				require.Nil(t, err, err)
				require.Len(t, fullspec.Apis, len(tt.result))
				for i := 0; i < len(tt.result); i++ {
					require.Equal(t, fullspec.Apis[i].Name, apis[tt.result[i]].Name)
					require.Equal(t, fullspec.Apis[i].Enabled, apis[tt.result[i]].Enabled)
				}
				keeper.SetSpec(ctx, fullspec)
			} else {
				require.NotNil(t, err)
			}
		})
	}
}
