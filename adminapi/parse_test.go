package adminapi

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseQuery(t *testing.T) {
	tests := []struct {
		name        string
		query       string
		want        Filters
		expectError bool
	}{
		{
			name:  "simple int value",
			query: "hostname=12345",
			want:  Filters{"hostname": 12345},
		},
		{
			name:  "float value",
			query: "hostname=12345",
			want:  Filters{"hostname": 12345},
		},
		{
			name:  "simple string value",
			query: `memory=10.2`,
			want:  Filters{"memory": 10.2},
		},
		{
			name:  "quoted string value",
			query: `description="quoted string"`,
			want:  Filters{"description": "quoted string"},
		},
		{
			name:  "bool value",
			query: "active=true",
			want:  Filters{"active": true},
		},
		{
			name:  "Regexp filter",
			query: "hostname=regexp(foo.*)",
			want:  Filters{"hostname": Filter{"Regexp": "foo.*"}},
		},
		{
			name:  "Not Empty filter",
			query: "hostname=not(empty())",
			want:  Filters{"hostname": Filter{"Not": Filter{"Empty": []interface{}{}}}},
		},
		{
			name:  "Any multi int",
			query: "game_world=Any(1 2 3)",
			want:  Filters{"game_world": Filter{"Any": []any{1, 2, 3}}},
		},
		{
			name:  "All with mixed types",
			query: "hypervisor=all(1 server true)",
			want:  Filters{"hypervisor": Filter{"All": []any{1, "server", true}}},
		},
		{
			name:  "Nested filters",
			query: "prop=not(any(Regexp(abc) 42))",
			want:  Filters{"prop": Filter{"Not": Filter{"Any": []any{Filter{"Regexp": "abc"}, 42}}}},
		},
		{
			name:  "Multiple fields",
			query: "hostname=foo id=123 active=false",
			want:  Filters{"hostname": "foo", "id": 123, "active": false},
		},
		// --- Broken/Invalid syntax cases ---
		{
			name:        "missing equals",
			query:       "hostnamefoo",
			expectError: true,
		},
		{
			name:        "empty",
			query:       "",
			expectError: true,
		},
		{
			name:        "only space",
			query:       "  ",
			expectError: true,
		},
		{
			name:        "unterminated parens",
			query:       "hostname=Any(1 2 3",
			expectError: true,
		},
		{
			name:        "bad filter format",
			query:       "id=StrangeFunc[1 2]",
			expectError: true,
		},
		{
			name:        "empty key",
			query:       "=123",
			expectError: true,
		},
		{
			name:  "missing value",
			query: "field=",
			want:  Filters{"field": ""},
		},
		{
			name:  "extra spaces",
			query: "  foo=4    bar=Not(  1 )  ",
			want:  Filters{"foo": 4, "bar": Filter{"Not": 1}},
		},
		// --- New test cases for uppercase/mixed-case functions and invalid function names ---
		{
			name:  "uppercase function name ALL",
			query: "meta=ALL(str1 str2)",
			want:  Filters{"meta": Filter{"All": []any{"str1", "str2"}}},
		},
		{
			name:  "mixed-case function name REGexp",
			query: "hostname=ReGExp(.*)",
			want:  Filters{"hostname": Filter{"Regexp": ".*"}},
		},
		{
			name:  "overlaps with capital letters",
			query: "field=OverLapS(A B C)",
			want:  Filters{"field": Filter{"Overlaps": []any{"A", "B", "C"}}},
		},
		{
			name:        "invalid function name",
			query:       "hostname=Nonexisting(.*)",
			expectError: true,
		},
		{
			name:        "unclosed function",
			query:       "hostname=regexp(sss",
			expectError: true,
		},
		{
			name:  "StartsWith function with normal case",
			query: "description=startsWith(abc)",
			want:  Filters{"description": Filter{"StartsWith": "abc"}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseQuery(tt.query)
			if tt.expectError {
				assert.Error(t, err, "expected error but got nil")
			} else {
				require.NoError(t, err, "unexpected error", err)
				assert.Equal(t, tt.want, got, "expected and actual output do not match")
			}
		})
	}
}

func BenchmarkParseQuery_Simple(b *testing.B) {
	query := "hostname=xxx.foo.bar"
	for b.Loop() {
		_, err := ParseQuery(query)
		if err != nil {
			b.Fatalf("Failed to parse query: %v", err)
		}
	}
}

func BenchmarkParseQuery_Complex(b *testing.B) {
	query := "server_type=not(empty) environment=any(production testing) memory=GreaterThan(100) foo=bar"
	for b.Loop() {
		_, err := ParseQuery(query)
		if err != nil {
			b.Fatalf("Failed to parse query: %v", err)
		}
	}
}
