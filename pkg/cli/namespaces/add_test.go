package namespaces

import (
	"testing"

	operatorUtils "github.com/percona/everest-operator/utils"
	"github.com/stretchr/testify/assert"
)

func TestParseNamespaceNames(t *testing.T) {
	t.Parallel()

	type tcase struct {
		name   string
		input  string
		output []string
	}

	tcases := []tcase{
		{
			name:   "empty string",
			input:  "",
			output: []string{},
		},
		{
			name:   "several empty strings",
			input:  "   ,   ,",
			output: []string{},
		},
		{
			name:   "correct",
			input:  "aaa,bbb,ccc",
			output: []string{"aaa", "bbb", "ccc"},
		},
		{
			name: "correct with spaces",
			input: `    aaa, bbb 
,ccc   `,
			output: []string{"aaa", "bbb", "ccc"},
		},
		{
			name:   "reserved system ns",
			input:  "everest-system",
			output: []string{"everest-system"},
		},
		{
			name:   "reserved system ns and empty ns",
			input:  "everest-system,    ",
			output: []string{"everest-system"},
		},
		{
			name:   "reserved monitoring ns",
			input:  "everest-monitoring",
			output: []string{"everest-monitoring"},
		},
		{
			name:   "reserved olm ns",
			input:  "everest-olm",
			output: []string{"everest-olm"},
		},
		{
			name:   "duplicated ns",
			input:  "aaa,bbb,aaa",
			output: []string{"aaa", "bbb"},
		},
		{
			name:   "name is too long",
			input:  "e1234567890123456789012345678901234567890123456789012345678901234567890,bbb",
			output: []string{"e1234567890123456789012345678901234567890123456789012345678901234567890", "bbb"},
		},
		{
			name:   "name starts with number",
			input:  "1aaa,bbb",
			output: []string{"1aaa", "bbb"},
		},
		{
			name:   "name contains special characters",
			input:  "aa12a,b$s",
			output: []string{"aa12a", "b$s"},
		},
	}

	for _, tc := range tcases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			output := ParseNamespaceNames(tc.input)
			assert.Equal(t, tc.output, output)
		})
	}
}

func TestValidateNamespaces(t *testing.T) {
	t.Parallel()

	type tcase struct {
		name  string
		input []string
		error error
	}

	tcases := []tcase{
		{
			name:  "empty list",
			input: []string{},
			error: ErrNamespaceListEmpty,
		},
		{
			name:  "empty string",
			input: []string{""},
			error: operatorUtils.ErrNameNotRFC1035Compatible("namespace name"),
		},
		{
			name:  "several empty strings",
			input: []string{"   ", "   "},
			error: operatorUtils.ErrNameNotRFC1035Compatible("namespace name"),
		},
		{
			name:  "correct",
			input: []string{"aaa", "bbb", "ccc"},
			error: nil,
		},
		{
			name:  "reserved system ns",
			input: []string{"everest-system"},
			error: ErrNamespaceReserved("everest-system"),
		},
		{
			name:  "reserved system ns and empty ns",
			input: []string{"everest-system", "    "},
			error: ErrNamespaceReserved("everest-system"),
		},
		{
			name:  "reserved monitoring ns",
			input: []string{"everest-monitoring"},
			error: ErrNamespaceReserved("everest-monitoring"),
		},
		{
			name:  "reserved olm ns",
			input: []string{"everest-olm"},
			error: ErrNamespaceReserved("everest-olm"),
		},
		{
			name:  "duplicated ns",
			input: []string{"aaa", "bbb", "aaa"},
			error: nil,
		},
		{
			name:  "name is too long",
			input: []string{"e1234567890123456789012345678901234567890123456789012345678901234567890", "bbb"},
			error: operatorUtils.ErrNameNotRFC1035Compatible("namespace name"),
		},
		{
			name:  "name starts with number",
			input: []string{"1aaa", "bbb"},
			error: operatorUtils.ErrNameNotRFC1035Compatible("namespace name"),
		},
		{
			name:  "name contains special characters",
			input: []string{"aa12a", "b$s"},
			error: operatorUtils.ErrNameNotRFC1035Compatible("namespace name"),
		},
	}

	for _, tc := range tcases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			err := validateNamespaceNames(tc.input)
			assert.Equal(t, tc.error, err)
			// assert.ElementsMatch(t, tc.output, output)
		})
	}
}
