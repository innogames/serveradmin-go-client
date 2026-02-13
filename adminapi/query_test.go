package adminapi

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSetAttributes(t *testing.T) {
	q := NewQuery(Filters{})

	// Default attributes
	assert.Equal(t, []string{"object_id", "hostname"}, q.restrictedAttributes)

	// SetAttributes replaces defaults
	q.SetAttributes("memory")
	assert.Equal(t, []string{"memory"}, q.restrictedAttributes)

	// SetAttributes with multiple arguments
	q.SetAttributes("hostname", "num_cpu", "memory")
	assert.Equal(t, []string{"hostname", "num_cpu", "memory"}, q.restrictedAttributes)
}

func TestAddAttributes(t *testing.T) {
	q := NewQuery(Filters{})

	// AddAttributes appends to defaults
	q.AddAttributes("memory")
	assert.Equal(t, []string{"object_id", "hostname", "memory"}, q.restrictedAttributes)

	// AddAttributes with multiple arguments
	q.AddAttributes("num_cpu", "state")
	assert.Equal(t, []string{"object_id", "hostname", "memory", "num_cpu", "state"}, q.restrictedAttributes)
}

func TestFilters(t *testing.T) {
	q := NewQuery(Filters{
		"hostname":   NotEmpty(),
		"num_cpu":    Regexp(".*GB"),
		"hypervisor": StartsWith("datacenter-x-"),
	})

	assert.Equal(t, Filters{
		"hostname":   Filter{"Not": Filter{"Empty": interface{}(nil)}},
		"num_cpu":    Filter{"Regexp": ".*GB"},
		"hypervisor": Filter{"StartsWith": "datacenter-x-"},
	}, q.filters)
}

func TestFromQuery(t *testing.T) {
	q, err := FromQuery("hostname=not(empty()) num_cpu=regexp(.*GB)")
	require.NoError(t, err)
	q.AddFilter("instance", 1)
	q.OrderBy("num_cpu")

	assert.Equal(t, Filters{
		"hostname": Filter{"Not": Filter{"Empty": []interface{}{}}},
		"num_cpu":  Filter{"Regexp": ".*GB"},
		"instance": 1,
	}, q.filters)
}

func TestFromQueryWithError(t *testing.T) {
	q, err := FromQuery("hostname=not(empty(")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "unmatched ( found")
	assert.Equal(t, Query{}, q, "query should be zero value on error")
}
