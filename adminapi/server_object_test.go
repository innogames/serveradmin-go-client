package adminapi

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSet(t *testing.T) {
	obj := &ServerObject{
		attributes: Attributes{"hostname": "old.local", "object_id": float64(1)},
		oldValues:  Attributes{},
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
		attributes: Attributes{"hostname": "test", "object_id": float64(1)},
		oldValues:  Attributes{},
	}

	err := obj.Set("nonexistent", "value")
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrUnknownAttribute)
}

func TestCommitState(t *testing.T) {
	// Consistent: no changes
	obj := &ServerObject{
		attributes: Attributes{"hostname": "test", "object_id": float64(1)},
		oldValues:  Attributes{},
	}
	assert.Equal(t, StateConsistent, obj.CommitState())

	// Changed: attribute modified
	obj.Set("hostname", "changed")
	assert.Equal(t, StateChanged, obj.CommitState())

	// Deleted
	obj2 := &ServerObject{
		attributes: Attributes{"hostname": "test", "object_id": float64(1)},
		oldValues:  Attributes{},
		deleted:    true,
	}
	assert.Equal(t, StateDeleted, obj2.CommitState())

	// Created: no object_id
	obj3 := &ServerObject{
		attributes: Attributes{"hostname": "test", "object_id": nil},
		oldValues:  Attributes{},
	}
	assert.Equal(t, StateCreated, obj3.CommitState())
}

func TestSerializeChanges(t *testing.T) {
	obj := &ServerObject{
		attributes: Attributes{"hostname": "new.local", "object_id": float64(42)},
		oldValues:  Attributes{"hostname": "old.local"},
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
		attributes: Attributes{
			"tags":      []any{"web", "new-tag"},
			"object_id": float64(42),
		},
		oldValues: Attributes{
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
		attributes: Attributes{"hostname": "original", "object_id": float64(1)},
		oldValues:  Attributes{},
	}

	obj.Set("hostname", "modified")
	assert.Equal(t, "modified", obj.GetString("hostname"))

	obj.Rollback()
	assert.Equal(t, "original", obj.GetString("hostname"))
	assert.Empty(t, obj.oldValues)
	assert.Equal(t, StateConsistent, obj.CommitState())
}

func TestRollback_Comprehensive(t *testing.T) {
	tests := []struct {
		name            string
		initialAttrs    Attributes
		initialOldVals  Attributes
		initialDeleted  bool
		modifications   func(*ServerObject)
		expectedAttrs   Attributes
		expectedDeleted bool
		expectedState   CommitState
	}{
		{
			name: "rollback single attribute change",
			initialAttrs: Attributes{
				"hostname":  "original.local",
				"object_id": float64(1),
			},
			initialOldVals: Attributes{},
			modifications: func(obj *ServerObject) {
				obj.Set("hostname", "modified.local")
			},
			expectedAttrs: Attributes{
				"hostname":  "original.local",
				"object_id": float64(1),
			},
			expectedDeleted: false,
			expectedState:   StateConsistent,
		},
		{
			name: "rollback multiple attribute changes",
			initialAttrs: Attributes{
				"hostname":    "original.local",
				"environment": "development",
				"object_id":   float64(2),
			},
			initialOldVals: Attributes{},
			modifications: func(obj *ServerObject) {
				obj.Set("hostname", "new.local")
				obj.Set("environment", "production")
			},
			expectedAttrs: Attributes{
				"hostname":    "original.local",
				"environment": "development",
				"object_id":   float64(2),
			},
			expectedDeleted: false,
			expectedState:   StateConsistent,
		},
		{
			name: "rollback multi-attribute (slice) changes",
			initialAttrs: Attributes{
				"tags":      []any{"web", "original"},
				"object_id": float64(3),
			},
			initialOldVals: Attributes{},
			modifications: func(obj *ServerObject) {
				obj.Set("tags", []string{"web", "modified", "new"})
			},
			expectedAttrs: Attributes{
				"tags":      []any{"web", "original"},
				"object_id": float64(3),
			},
			expectedDeleted: false,
			expectedState:   StateConsistent,
		},
		{
			name: "rollback deleted object",
			initialAttrs: Attributes{
				"hostname":  "test.local",
				"object_id": float64(4),
			},
			initialOldVals: Attributes{},
			modifications: func(obj *ServerObject) {
				obj.Delete()
			},
			expectedAttrs: Attributes{
				"hostname":  "test.local",
				"object_id": float64(4),
			},
			expectedDeleted: false,
			expectedState:   StateConsistent,
		},
		{
			name: "rollback deleted object with attribute changes",
			initialAttrs: Attributes{
				"hostname":  "original.local",
				"object_id": float64(5),
			},
			initialOldVals: Attributes{},
			modifications: func(obj *ServerObject) {
				obj.Set("hostname", "modified.local")
				obj.Delete()
			},
			expectedAttrs: Attributes{
				"hostname":  "original.local",
				"object_id": float64(5),
			},
			expectedDeleted: false,
			expectedState:   StateConsistent,
		},
		{
			name: "rollback with no changes",
			initialAttrs: Attributes{
				"hostname":  "unchanged.local",
				"object_id": float64(6),
			},
			initialOldVals: Attributes{},
			modifications:  func(_ *ServerObject) {},
			expectedAttrs: Attributes{
				"hostname":  "unchanged.local",
				"object_id": float64(6),
			},
			expectedDeleted: false,
			expectedState:   StateConsistent,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			obj := &ServerObject{
				attributes: tt.initialAttrs,
				oldValues:  tt.initialOldVals,
				deleted:    tt.initialDeleted,
			}

			// Apply modifications
			tt.modifications(obj)

			// Rollback
			obj.Rollback()

			// Verify attributes are restored
			assert.Equal(t, tt.expectedAttrs, obj.attributes,
				"attributes should be restored to original values")

			// Verify oldValues is cleared
			assert.Empty(t, obj.oldValues, "oldValues should be empty after rollback")

			// Verify deleted flag is reset
			assert.Equal(t, tt.expectedDeleted, obj.deleted,
				"deleted flag should be reset")

			// Verify commit state
			assert.Equal(t, tt.expectedState, obj.CommitState(),
				"commit state should be consistent after rollback")
		})
	}
}

func TestGetMulti(t *testing.T) {
	tests := []struct {
		name     string
		attrs    Attributes
		key      string
		expected MultiAttr
	}{
		{
			name:     "[]any with strings",
			attrs:    Attributes{"tags": []any{"web", "prod"}},
			key:      "tags",
			expected: MultiAttr{"web", "prod"},
		},
		{
			name:     "[]string",
			attrs:    Attributes{"tags": []string{"web", "prod"}},
			key:      "tags",
			expected: MultiAttr{"web", "prod"},
		},
		{
			name:     "MultiAttr directly",
			attrs:    Attributes{"tags": MultiAttr{"web"}},
			key:      "tags",
			expected: MultiAttr{"web"},
		},
		{
			name:     "missing attribute",
			attrs:    Attributes{"hostname": "test"},
			key:      "tags",
			expected: MultiAttr{},
		},
		{
			name:     "nil value",
			attrs:    Attributes{"tags": nil},
			key:      "tags",
			expected: MultiAttr{},
		},
		{
			name:     "non-slice value (string)",
			attrs:    Attributes{"tags": "not-a-slice"},
			key:      "tags",
			expected: MultiAttr{},
		},
		{
			name:     "non-slice value (int)",
			attrs:    Attributes{"tags": 42},
			key:      "tags",
			expected: MultiAttr{},
		},
		{
			name:     "[]any with mixed types keeps only strings",
			attrs:    Attributes{"tags": []any{"web", 42, true, "prod"}},
			key:      "tags",
			expected: MultiAttr{"web", "prod"},
		},
		{
			name:     "empty []any",
			attrs:    Attributes{"tags": []any{}},
			key:      "tags",
			expected: MultiAttr{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			obj := &ServerObject{
				attributes: tt.attrs,
				oldValues:  Attributes{},
			}
			result := obj.GetMulti(tt.key)
			assert.Equal(t, tt.expected, result)
		})
	}
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
		oldValues:  Attributes{},
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
		oldValues:  Attributes{},
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
