package adminapi

import (
	"testing"

	"github.com/stretchr/testify/assert"
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
