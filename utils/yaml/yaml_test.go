package yaml

import (
	"testing"

	"github.com/stretchr/testify/require"
)

type mockStruct struct {
	Num  int
	Str  string
	Bool bool
}

var mockDefault = mockStruct{
	Num:  1,
	Str:  "string",
	Bool: true,
}

func TestDecode(t *testing.T) {
	var result mockStruct

	// decode empty string
	input := ""
	err := Decode(input, "key", &result, nil, nil, nil)
	require.Error(t, err)

	// decode no key
	input = `
nokey:
  num: 1
  str: string
  bool: True
`
	err = Decode(input, "key", &result, nil, nil, nil)
	require.Error(t, err)

	// decode good key
	input = `
key:
  num: 1
  str: string
  bool: True
`
	err = Decode(input, "key", &result, nil, nil, nil)
	require.NoError(t, err)
	require.Equal(t, mockDefault, result)

	// decode nested key
	input = `
nested:
  key:
    num: 1
    str: string
    bool: True
`
	err = Decode(input, "nested.key", &result, nil, nil, nil)
	require.NoError(t, err)
	require.Equal(t, mockDefault, result)

	// decode empty value
	input = `
key:
`
	err = Decode(input, "key", &result, nil, nil, nil)
	require.Error(t, err)

	// decode leaf value
	input = `
key: nothing
`
	err = Decode(input, "key", &result, nil, nil, nil)
	require.Error(t, err)
}

func TestDecodeMissing(t *testing.T) {
	var result mockStruct
	var unset  []string

	// decode nothing missing
	input := `
key:
  num: 1
  str: string
  bool: True
`
	err := Decode(input, "key", &result, nil, &unset, nil)
	require.NoError(t, err)
	require.Equal(t, 0, len(unset))
	require.Equal(t, mockDefault, result)

	// decode 1 missing field
	input = `
key:
  num: 1
  str: string
`
	err = Decode(input, "key", &result, nil, &unset, nil)
	require.NoError(t, err)
	require.Equal(t, 1, len(unset))
	require.Equal(t, "Bool", unset[0])

	// decode 2 missing fields
	input = `
key:
  num: 1
`
	err = Decode(input, "key", &result, nil, &unset, nil)
	require.NoError(t, err)
	require.Equal(t, 2, len(unset))
}

func TestDecodeExtra(t *testing.T) {
	var result mockStruct
	var unused []string

	// decode nothing extra
	input := `
key:
  num: 1
  str: string
  bool: True
`
	err := Decode(input, "key", &result, nil, nil, &unused)
	require.NoError(t, err)
	require.Equal(t, 0, len(unused))
	require.Equal(t, mockDefault, result)

	// decode 1 extra field
	input = `
key:
  num: 1
  str: string
  bool: True
  blah: extra
`
	err = Decode(input, "key", &result, nil, nil, &unused)
	require.NoError(t, err)
	require.Equal(t, 1, len(unused))
	require.Equal(t, "blah", unused[0])

	// decode 2 extra fields
	input = `
key:
  num: 1
  str: string
  bool: True
  blee: extra
  blah:
    extra: extra
`
	err = Decode(input, "key", &result, nil, nil, &unused)
	require.NoError(t, err)
	require.Equal(t, 2, len(unused))
}

func TestSetDefault(t *testing.T) {
	var result mockStruct
	var unset  []string

	// confirm not equal at start
	require.NotEqual(t, mockDefault, result)

	// decode 2 missing
	input := `
key:
  num: 1
  #str: string
  #bool: True
`
	err := Decode(input, "key", &result, nil, &unset, nil)
	require.NoError(t, err)
	require.Equal(t, 2, len(unset))
	// still not equal because of unset fields
	require.NotEqual(t, mockDefault, result)

	defs := map[string]interface{}{
		"str": "string",
	}

	err = SetDefaultValues(defs, &result)
	require.NoError(t, err)
	require.Equal(t, false, result.Bool)
	result.Bool = true
	require.Equal(t, mockDefault, result)
}
