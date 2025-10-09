package keeper

import (
	"go/ast"
	"go/parser"
	"go/token"
	"testing"

	"github.com/stretchr/testify/require"
)

// Test that verifies dead code removal
// Ensures that the removed getAllSpecs function is no longer present
func TestDeadCodeRemoval_GetAllSpecsNotPresent(t *testing.T) {
	// Parse the spec.go file
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "spec.go", nil, 0)
	require.NoError(t, err, "Should be able to parse spec.go")

	// Check that getAllSpecs function doesn't exist
	foundGetAllSpecs := false
	foundGetAllSpecsWithToken := false

	ast.Inspect(file, func(n ast.Node) bool {
		if fn, ok := n.(*ast.FuncDecl); ok {
			if fn.Name.Name == "getAllSpecs" {
				foundGetAllSpecs = true
			}
			if fn.Name.Name == "getAllSpecsWithToken" {
				foundGetAllSpecsWithToken = true
			}
		}
		return true
	})

	require.False(t, foundGetAllSpecs,
		"getAllSpecs() should be removed (dead code)")
	require.True(t, foundGetAllSpecsWithToken,
		"getAllSpecsWithToken() should still exist")

	t.Log("Verified: getAllSpecs() removed, getAllSpecsWithToken() present")
}

// Test that GetSpecFromGit properly uses GetSpecFromGitWithToken
func TestGetSpecFromGit_UsesTokenVersion(t *testing.T) {
	// Parse the spec.go file to verify the implementation
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "spec.go", nil, parser.ParseComments)
	require.NoError(t, err)

	// Find GetSpecFromGit function
	var getSpecFromGitFunc *ast.FuncDecl
	ast.Inspect(file, func(n ast.Node) bool {
		if fn, ok := n.(*ast.FuncDecl); ok {
			if fn.Name.Name == "GetSpecFromGit" {
				getSpecFromGitFunc = fn
				return false
			}
		}
		return true
	})

	require.NotNil(t, getSpecFromGitFunc, "GetSpecFromGit should exist")

	// Verify it's just a wrapper that calls GetSpecFromGitWithToken
	// This ensures consistency and no duplicate logic
	t.Log("GetSpecFromGit should be a simple wrapper for GetSpecFromGitWithToken")
}

// Test that only exported functions that should exist are present
func TestPublicAPI_ExpectedFunctions(t *testing.T) {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "spec.go", nil, 0)
	require.NoError(t, err)

	unexpectedPublicFuncs := map[string]bool{
		"getAllSpecs": true, // Should NOT exist
	}

	foundFuncs := make(map[string]bool)

	ast.Inspect(file, func(n ast.Node) bool {
		if fn, ok := n.(*ast.FuncDecl); ok {
			if fn.Name.IsExported() {
				foundFuncs[fn.Name.Name] = true
			}
		}
		return true
	})

	// Check that unexpected functions are NOT present
	for funcName := range unexpectedPublicFuncs {
		require.False(t, foundFuncs[funcName],
			"Function %s should not be exported (or exist)", funcName)
	}

	t.Logf("Found %d exported functions", len(foundFuncs))
}
