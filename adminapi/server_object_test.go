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

func TestSetOnDeleted(t *testing.T) {
	obj := &ServerObject{
		attributes: map[string]any{"hostname": "test", "object_id": float64(1)},
		oldValues:  map[string]any{},
		deleted:    true,
	}

	err := obj.Set("hostname", "new")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "deleted")
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
