package adminapi

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSet(t *testing.T) {
	obj := &ServerObject{
		attributes: map[string]any{"hostname": "old.local", "object_id": float64(1)},
		oldValues:  map[string]any{},
	}

	err := obj.Set("hostname", "new.local")
	require.NoError(t, err)
	assert.Equal(t, "new.local", obj.GetString("hostname"))
	assert.Equal(t, "old.local", obj.oldValues["hostname"])

	// Second set should not overwrite oldValues
	err = obj.Set("hostname", "newer.local")
	require.NoError(t, err)
	assert.Equal(t, "old.local", obj.oldValues["hostname"])
}

func TestSetNonexistent(t *testing.T) {
	obj := &ServerObject{
		attributes: map[string]any{"hostname": "test", "object_id": float64(1)},
		oldValues:  map[string]any{},
	}

	err := obj.Set("nonexistent", "value")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "does not exist")
}

func TestCommitState(t *testing.T) {
	// Consistent: no changes
	obj := &ServerObject{
		attributes: map[string]any{"hostname": "test", "object_id": float64(1)},
		oldValues:  map[string]any{},
	}
	assert.Equal(t, "consistent", obj.CommitState())

	// Changed: attribute modified
	obj.Set("hostname", "changed")
	assert.Equal(t, "changed", obj.CommitState())

	// Deleted
	obj2 := &ServerObject{
		attributes: map[string]any{"hostname": "test", "object_id": float64(1)},
		oldValues:  map[string]any{},
		deleted:    true,
	}
	assert.Equal(t, "deleted", obj2.CommitState())

	// Created: no object_id
	obj3 := &ServerObject{
		attributes: map[string]any{"hostname": "test", "object_id": nil},
		oldValues:  map[string]any{},
	}
	assert.Equal(t, "created", obj3.CommitState())
}

func TestSerializeChanges(t *testing.T) {
	obj := &ServerObject{
		attributes: map[string]any{"hostname": "new.local", "object_id": float64(42)},
		oldValues:  map[string]any{"hostname": "old.local"},
	}

	changes := obj.serializeChanges()
	assert.Equal(t, 42, changes["object_id"])

	hostChange := changes["hostname"].(map[string]any)
	assert.Equal(t, "update", hostChange["action"])
	assert.Equal(t, "old.local", hostChange["old"])
	assert.Equal(t, "new.local", hostChange["new"])
}

func TestSerializeChangesMulti(t *testing.T) {
	obj := &ServerObject{
		attributes: map[string]any{
			"tags":      []any{"web", "new-tag"},
			"object_id": float64(42),
		},
		oldValues: map[string]any{
			"tags": []any{"web", "old-tag"},
		},
	}

	changes := obj.serializeChanges()
	tagChange := changes["tags"].(map[string]any)
	assert.Equal(t, "multi", tagChange["action"])
	assert.Contains(t, tagChange["add"], "new-tag")
	assert.Contains(t, tagChange["remove"], "old-tag")
}

func TestRollback(t *testing.T) {
	obj := &ServerObject{
		attributes: map[string]any{"hostname": "original", "object_id": float64(1)},
		oldValues:  map[string]any{},
	}

	obj.Set("hostname", "modified")
	assert.Equal(t, "modified", obj.GetString("hostname"))

	obj.Rollback()
	assert.Equal(t, "original", obj.GetString("hostname"))
	assert.Empty(t, obj.oldValues)
	assert.Equal(t, "consistent", obj.CommitState())
}

func TestSetMultiAttribute_WithStringSlice(t *testing.T) {
	// Simulate a fetched object with a multi-attribute
	// (JSON decoding produces []any for arrays)
	attributes := map[string]any{
		"object_id": float64(12345),
		"hostname":  "test.example.com",
		"dns_txt":   []any{"existing", "values"},
	}

	obj := &ServerObject{
		attributes: attributes,
		oldValues:  map[string]any{},
	}

	// User sets the attribute using []string (common usage)
	err := obj.Set("dns_txt", []string{"new", "values"})
	require.NoError(t, err)

	// Verify oldValues captured the original
	assert.Equal(t, []any{"existing", "values"}, obj.oldValues["dns_txt"])

	// Serialize changes
	changes := obj.serializeChanges()

	// Should use "multi" action, not "update"
	dnsChange, ok := changes["dns_txt"].(map[string]any)
	require.True(t, ok, "dns_txt change should be a map")

	assert.Equal(t, "multi", dnsChange["action"],
		"Multi-attribute should use 'multi' action even with []string, not 'update'")

	// Verify correct add/remove sets
	add := dnsChange["add"].([]any)
	remove := dnsChange["remove"].([]any)

	assert.ElementsMatch(t, []any{"new"}, add)
	assert.ElementsMatch(t, []any{"existing"}, remove)
}

func TestSetMultiAttribute_WithIntSlice(t *testing.T) {
	attributes := map[string]any{
		"object_id": float64(12345),
		"ports":     []any{80, 443},
	}

	obj := &ServerObject{
		attributes: attributes,
		oldValues:  map[string]any{},
	}

	// User passes []int
	err := obj.Set("ports", []int{443, 8080})
	require.NoError(t, err)

	changes := obj.serializeChanges()
	portsChange := changes["ports"].(map[string]any)

	assert.Equal(t, "multi", portsChange["action"])
	assert.ElementsMatch(t, []any{8080}, portsChange["add"])
	assert.ElementsMatch(t, []any{80}, portsChange["remove"])
}

func TestToAnySlice_VariousTypes(t *testing.T) {
	tests := []struct {
		name     string
		input    any
		expected []any
	}{
		{
			name:     "already []any",
			input:    []any{1, 2, 3},
			expected: []any{1, 2, 3},
		},
		{
			name:     "[]string",
			input:    []string{"a", "b", "c"},
			expected: []any{"a", "b", "c"},
		},
		{
			name:     "[]int",
			input:    []int{1, 2, 3},
			expected: []any{1, 2, 3},
		},
		{
			name:     "[]interface{} with mixed types",
			input:    []interface{}{"str", 42, true},
			expected: []any{"str", 42, true},
		},
		{
			name:     "not a slice",
			input:    "string",
			expected: nil,
		},
		{
			name:     "nil",
			input:    nil,
			expected: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := toAnySlice(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}
