package lavasession

import (
	"testing"

	spectypes "github.com/lavanet/lava/v4/x/spec/types"
	"github.com/stretchr/testify/require"
)

func TestRouterKey_SetExtensions(t *testing.T) {
	rk := NewRouterKey([]string{"ext1", "ext2", "ext1"})
	require.Equal(t, "|ext1|ext2|", rk.String())

	rk.SetExtensions([]string{"ext3", "ext2"})
	require.Equal(t, "|ext2|ext3|", rk.String())
}

func TestRouterKey_NewRouterKeyFromExtensions(t *testing.T) {
	rk := NewRouterKeyFromExtensions([]*spectypes.Extension{
		{Name: "ext1"},
		{Name: "ext2"},
		{Name: "ext3"},
	})
	require.Equal(t, "|ext1|ext2|ext3|", rk.String())
}

func TestRouterKey_HasExtension(t *testing.T) {
	rk := NewRouterKey([]string{"ext1", "ext2"})
	require.True(t, rk.HasExtension("ext1"))
	require.False(t, rk.HasExtension("ext3"))
}

func TestRouterKey_ApplyMethodsRoute(t *testing.T) {
	rk := NewRouterKey([]string{})
	rk.ApplyMethodsRoute(42)
	require.Equal(t, "||method-route:42|", rk.String())
}

func TestRouterKey_ApplyInternalPath(t *testing.T) {
	rk := NewRouterKey([]string{})
	rk.ApplyInternalPath("/x")
	require.Equal(t, "||internal-path:/x|", rk.String())
}

func TestRouterKey_String_NoExtensionsNoRouteNoPath(t *testing.T) {
	rk := NewRouterKey([]string{})
	require.Equal(t, "||", rk.String())
}

func TestRouterKey_String_WithExtensionsNoRouteNoPath(t *testing.T) {
	rk := NewRouterKey([]string{"ext2", "ext1"})
	require.Equal(t, "|ext1|ext2|", rk.String())
}

func TestRouterKey_String_WithExtensionsAndRouteNoPath(t *testing.T) {
	rk := NewRouterKey([]string{"ext1", "ext2"})
	rk.ApplyMethodsRoute(42)
	require.Equal(t, "|ext1|ext2|method-route:42|", rk.String())
}

func TestRouterKey_String_WithExtensionsRouteAndPath(t *testing.T) {
	rk := NewRouterKey([]string{"ext1", "ext2"})
	rk.ApplyMethodsRoute(42)
	rk.ApplyInternalPath("/x")
	require.Equal(t, "|ext1|ext2|method-route:42|internal-path:/x|", rk.String())
}

func TestRouterKey_String_NoExtensionsWithRouteAndPath(t *testing.T) {
	rk := NewRouterKey([]string{})
	rk.ApplyMethodsRoute(42)
	rk.ApplyInternalPath("/x")
	require.Equal(t, "||method-route:42|internal-path:/x|", rk.String())
}

func TestRouterKey_String_WithPathNoRouteNoExtensions(t *testing.T) {
	rk := NewRouterKey([]string{})
	rk.ApplyInternalPath("/another/path")
	require.Equal(t, "||internal-path:/another/path|", rk.String())
}

func TestGetEmptyRouterKey(t *testing.T) {
	rk := GetEmptyRouterKey()
	require.Equal(t, "||", rk.String())
}

func TestRouterKey_SetExtensions_EmptySlice(t *testing.T) {
	rk := NewRouterKey([]string{})
	rk.SetExtensions([]string{})
	require.Equal(t, "||", rk.String())
}
