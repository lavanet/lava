package decoder

import (
	"strings"
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

var mockDefaultYaml = `  num: 1
  str: string
  bool: True
`

func indent(yaml string) string {
	lines := strings.Split(yaml, "\n")
	for i := range lines {
		lines[i] = "  " + lines[i]
	}
	return strings.Join(lines, "\n")
}

func comment(yaml string, from, count int) string {
	lines := strings.Split(yaml, "\n")
	for i := range lines {
		if i >= from && i < from+count {
			lines[i] = "#" + lines[i]
		}
	}
	return strings.Join(lines, "\n")
}

const keyStr = "key:\n"

func TestDecode(t *testing.T) {
	var result mockStruct

	// decode empty string
	input := ""
	err := Decode(input, "key", &result, nil, nil, nil)
	require.Error(t, err)

	// decode no key
	input = "nokey:\n" + mockDefaultYaml
	err = Decode(input, "key", &result, nil, nil, nil)
	require.Error(t, err)

	// decode good key
	input = keyStr + mockDefaultYaml
	err = Decode(input, "key", &result, nil, nil, nil)
	require.NoError(t, err)
	require.Equal(t, mockDefault, result)

	// decode nested key
	input = "nested:\n" + "  " + keyStr + indent(mockDefaultYaml)
	err = Decode(input, "nested.key", &result, nil, nil, nil)
	require.NoError(t, err)
	require.Equal(t, mockDefault, result)

	// decode empty value
	input = keyStr
	err = Decode(input, "key", &result, nil, nil, nil)
	require.Error(t, err)

	// decode leaf value
	input = "key: nothing\n"
	err = Decode(input, "key", &result, nil, nil, nil)
	require.Error(t, err)
}

func TestDecodeMissing(t *testing.T) {
	var result mockStruct
	var unset []string

	// decode nothing missing
	input := keyStr + mockDefaultYaml
	err := Decode(input, "key", &result, nil, &unset, nil)
	require.NoError(t, err)
	require.Equal(t, 0, len(unset))
	require.Equal(t, mockDefault, result)

	// decode 1 missing field
	input = keyStr + "  num: 1\n" + "  str: string\n"
	err = Decode(input, "key", &result, nil, &unset, nil)
	require.NoError(t, err)
	require.Equal(t, 1, len(unset))
	require.Equal(t, "Bool", unset[0])

	// decode 2 missing fields
	input = keyStr + "  num: 1\n"
	err = Decode(input, "key", &result, nil, &unset, nil)
	require.NoError(t, err)
	require.Equal(t, 2, len(unset))
}

func TestDecodeExtra(t *testing.T) {
	var result mockStruct
	var unused []string

	// decode nothing extra
	input := keyStr + mockDefaultYaml
	err := Decode(input, "key", &result, nil, nil, &unused)
	require.NoError(t, err)
	require.Equal(t, 0, len(unused))
	require.Equal(t, mockDefault, result)

	// decode 1 extra field
	input += "  blee: extra\n"
	err = Decode(input, "key", &result, nil, nil, &unused)
	require.NoError(t, err)
	require.Equal(t, 1, len(unused))
	require.Equal(t, "blee", unused[0])

	// decode 2 extra fields
	input += "  blah:\n" + "    extra: extra\n"
	err = Decode(input, "key", &result, nil, nil, &unused)
	require.NoError(t, err)
	require.Equal(t, 2, len(unused))
}

func TestDecodeArray(t *testing.T) {
	var results []mockStruct

	mock1 := mockStruct{
		Num:  1,
		Str:  "string",
		Bool: true,
	}
	mock2 := mockStruct{
		Num:  2,
		Str:  "string",
		Bool: true,
	}

	mock1yaml := `  - num: 1
    str: string
    bool: True
`
	mock2yaml := `  - num: 2
    str: string
    bool: True
`
	input := keyStr + mock1yaml + mock2yaml
	err := Decode(input, "key", &results, nil, nil, nil)
	require.NoError(t, err)
	require.Equal(t, []mockStruct{mock1, mock2}, results)
}

func TestSetDefault(t *testing.T) {
	var result mockStruct
	var unset []string

	// confirm not equal at start
	require.NotEqual(t, mockDefault, result)

	// decode 2 missing
	input := keyStr + comment(mockDefaultYaml, 1, 2)
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
