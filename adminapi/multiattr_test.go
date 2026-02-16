package adminapi

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMultiAttr_Add(t *testing.T) {
	tests := []struct {
		name     string
		initial  MultiAttr
		toAdd    []string
		expected MultiAttr
	}{
		{
			name:     "add single element to existing",
			initial:  MultiAttr{"web", "prod"},
			toAdd:    []string{"api"},
			expected: MultiAttr{"web", "prod", "api"},
		},
		{
			name:     "add multiple elements at once",
			initial:  MultiAttr{"web"},
			toAdd:    []string{"api", "prod", "monitoring"},
			expected: MultiAttr{"web", "api", "prod", "monitoring"},
		},
		{
			name:     "add duplicates (should skip)",
			initial:  MultiAttr{"web", "prod"},
			toAdd:    []string{"web", "api", "prod"},
			expected: MultiAttr{"web", "prod", "api"},
		},
		{
			name:     "add to empty MultiAttr",
			initial:  MultiAttr{},
			toAdd:    []string{"web", "prod"},
			expected: MultiAttr{"web", "prod"},
		},
		{
			name:     "add nothing",
			initial:  MultiAttr{"web"},
			toAdd:    []string{},
			expected: MultiAttr{"web"},
		},
		{
			name:     "add empty string",
			initial:  MultiAttr{"web"},
			toAdd:    []string{""},
			expected: MultiAttr{"web", ""},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := tt.initial
			m.Add(tt.toAdd...)
			assert.Equal(t, tt.expected, m)
		})
	}
}

func TestMultiAttr_AddToNil(t *testing.T) {
	var m MultiAttr
	m.Add("web", "prod")
	assert.Equal(t, MultiAttr{"web", "prod"}, m)
}

func TestMultiAttr_Delete(t *testing.T) {
	tests := []struct {
		name     string
		initial  MultiAttr
		toDelete string
		expected MultiAttr
	}{
		{
			name:     "delete existing element",
			initial:  MultiAttr{"web", "prod", "api"},
			toDelete: "prod",
			expected: MultiAttr{"web", "api"},
		},
		{
			name:     "delete non-existent element (no-op)",
			initial:  MultiAttr{"web", "prod"},
			toDelete: "api",
			expected: MultiAttr{"web", "prod"},
		},
		{
			name:     "delete all occurrences of duplicate",
			initial:  MultiAttr{"web", "prod", "web", "api"},
			toDelete: "web",
			expected: MultiAttr{"prod", "api"},
		},
		{
			name:     "delete from empty MultiAttr (no-op)",
			initial:  MultiAttr{},
			toDelete: "web",
			expected: MultiAttr{},
		},
		{
			name:     "delete last element",
			initial:  MultiAttr{"web"},
			toDelete: "web",
			expected: MultiAttr{},
		},
		{
			name:     "delete empty string",
			initial:  MultiAttr{"web", "", "prod"},
			toDelete: "",
			expected: MultiAttr{"web", "prod"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := tt.initial
			m.Delete(tt.toDelete)
			assert.Equal(t, tt.expected, m)
		})
	}
}

func TestMultiAttr_Clear(t *testing.T) {
	tests := []struct {
		name    string
		initial MultiAttr
	}{
		{
			name:    "clear non-empty MultiAttr",
			initial: MultiAttr{"web", "prod", "api"},
		},
		{
			name:    "clear already empty MultiAttr",
			initial: MultiAttr{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := tt.initial
			m.Clear()
			assert.Empty(t, m)
			assert.Equal(t, MultiAttr{}, m)
		})
	}
}

func TestMultiAttr_ClearNil(t *testing.T) {
	var m MultiAttr
	m.Clear() // Should not panic
	assert.Empty(t, m)
	assert.Equal(t, MultiAttr{}, m)
}

func TestMultiAttr_Contains(t *testing.T) {
	tests := []struct {
		name     string
		m        MultiAttr
		elem     string
		expected bool
	}{
		{
			name:     "contains existing element",
			m:        MultiAttr{"web", "prod", "api"},
			elem:     "prod",
			expected: true,
		},
		{
			name:     "contains non-existent element",
			m:        MultiAttr{"web", "prod"},
			elem:     "api",
			expected: false,
		},
		{
			name:     "contains on empty MultiAttr",
			m:        MultiAttr{},
			elem:     "web",
			expected: false,
		},
		{
			name:     "contains on nil MultiAttr",
			m:        nil,
			elem:     "web",
			expected: false,
		},
		{
			name:     "contains empty string",
			m:        MultiAttr{"web", "", "prod"},
			elem:     "",
			expected: true,
		},
		{
			name:     "case sensitive",
			m:        MultiAttr{"web", "PROD"},
			elem:     "prod",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.m.Contains(tt.elem)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestMultiAttr_Integration(t *testing.T) {
	// Simulate full workflow with ServerObject
	obj := &ServerObject{
		attributes: Attributes{
			"tags":      []any{"web", "old-tag"},
			"object_id": float64(42),
		},
		oldValues: Attributes{},
	}

	// Get tags as MultiAttr
	tags := obj.GetMulti("tags")

	// Manipulate with MultiAttr methods
	tags.Add("api", "prod")
	tags.Delete("old-tag")

	// Verify state before Set
	assert.True(t, tags.Contains("web"))
	assert.True(t, tags.Contains("api"))
	assert.True(t, tags.Contains("prod"))
	assert.False(t, tags.Contains("old-tag"))

	// Set back to ServerObject
	err := obj.Set("tags", []string(tags))
	require.NoError(t, err)

	// Serialize changes and verify correct add/remove sets
	changes := obj.serializeChanges()

	tagChange, ok := changes["tags"].(map[string]any)
	require.True(t, ok, "tags change should be a map")
	assert.Equal(t, "multi", tagChange["action"])

	add := tagChange["add"].([]any)
	remove := tagChange["remove"].([]any)

	// Should add: api, prod
	// Should remove: old-tag
	// Should keep: web (unchanged)
	assert.ElementsMatch(t, []any{"api", "prod"}, add)
	assert.ElementsMatch(t, []any{"old-tag"}, remove)
}

func TestMultiAttr_ChainedOperations(t *testing.T) {
	// Test complex sequences of operations
	m := MultiAttr{"web", "prod"}

	m.Add("api")
	assert.ElementsMatch(t, []string{"web", "prod", "api"}, m)

	m.Add("web") // Duplicate, should not add
	assert.Len(t, m, 3)

	m.Delete("prod")
	assert.ElementsMatch(t, []string{"web", "api"}, m)

	m.Add("monitoring", "logging")
	assert.ElementsMatch(t, []string{"web", "api", "monitoring", "logging"}, m)

	m.Clear()
	assert.Empty(t, m)

	m.Add("new")
	assert.Equal(t, MultiAttr{"new"}, m)
}

func TestMultiAttr_OrderPreservation(t *testing.T) {
	// While we use set semantics, order should be preserved for existing elements
	m := MultiAttr{"first", "second", "third"}

	m.Add("fourth")
	assert.Equal(t, MultiAttr{"first", "second", "third", "fourth"}, m)

	m.Delete("second")
	assert.Equal(t, MultiAttr{"first", "third", "fourth"}, m)
}
