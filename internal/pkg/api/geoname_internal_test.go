package api

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestFts5PrefixQuery(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"Bern", `"Bern"*`},
		{"New York", `"New" "York"*`},
		{"  spaces  ", `"spaces"*`},
		{`quo"te`, `"quo""te"*`},
		{"OR NOT", `"OR" "NOT"*`},
		{"AND", `"AND"*`},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			require.Equal(t, tt.want, fts5PrefixQuery(tt.input))
		})
	}
}
