package adminapi

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCommitSingle(t *testing.T) {
	var receivedBody commitRequest

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		json.Unmarshal(body, &receivedBody)

		w.WriteHeader(200)
		w.Write([]byte(`{"status": "success", "commit_id": 123}`))
	}))
	defer server.Close()

	resetConfig()
	t.Setenv("SERVERADMIN_TOKEN", "testtoken")
	t.Setenv("SERVERADMIN_BASE_URL", server.URL)

	obj := &ServerObject{
		attributes: map[string]any{"hostname": "new.local", "object_id": float64(42)},
		oldValues:  map[string]any{"hostname": "old.local"},
	}

	commitID, err := obj.Commit()
	require.NoError(t, err)
	assert.Equal(t, 123, commitID)

	// Verify payload
	assert.Len(t, receivedBody.Changed, 1)
	assert.Empty(t, receivedBody.Created)
	assert.Empty(t, receivedBody.Deleted)

	// State should be reset after commit
	assert.Equal(t, "consistent", obj.CommitState())
	assert.Empty(t, obj.oldValues)
}

func TestCommitResultSet(t *testing.T) {
	var receivedBody commitRequest

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		json.Unmarshal(body, &receivedBody)

		w.WriteHeader(200)
		w.Write([]byte(`{"status": "success", "commit_id": 456}`))
	}))
	defer server.Close()

	resetConfig()
	t.Setenv("SERVERADMIN_TOKEN", "testtoken")
	t.Setenv("SERVERADMIN_BASE_URL", server.URL)

	objects := ServerObjects{
		{
			attributes: map[string]any{"hostname": "changed.local", "object_id": float64(1)},
			oldValues:  map[string]any{"hostname": "orig1.local"},
		},
		{
			attributes: map[string]any{"hostname": "unchanged.local", "object_id": float64(2)},
			oldValues:  map[string]any{},
		},
		{
			attributes: map[string]any{"hostname": "deleted.local", "object_id": float64(3)},
			oldValues:  map[string]any{},
			deleted:    true,
		},
	}

	commitID, err := objects.Commit()
	require.NoError(t, err)
	assert.Equal(t, 456, commitID)

	assert.Len(t, receivedBody.Changed, 1)
	assert.Len(t, receivedBody.Deleted, 1)
	assert.Empty(t, receivedBody.Created)
}

func TestServerObjectsSetSuccess(t *testing.T) {
	objects := ServerObjects{
		{
			attributes: map[string]any{"hostname": "server1", "object_id": float64(1)},
			oldValues:  map[string]any{},
		},
		{
			attributes: map[string]any{"hostname": "server2", "object_id": float64(2)},
			oldValues:  map[string]any{},
		},
	}

	err := objects.Set("hostname", "updated")
	require.NoError(t, err)

	assert.Equal(t, "updated", objects[0].GetString("hostname"))
	assert.Equal(t, "updated", objects[1].GetString("hostname"))
	assert.Equal(t, "server1", objects[0].oldValues["hostname"])
	assert.Equal(t, "server2", objects[1].oldValues["hostname"])
}

func TestServerObjectsSetAllErrors(t *testing.T) {
	objects := ServerObjects{
		{
			attributes: map[string]any{"hostname": "server1", "object_id": float64(1)},
			oldValues:  map[string]any{},
		},
		{
			attributes: map[string]any{"hostname": "server2", "object_id": float64(2)},
			oldValues:  map[string]any{},
		},
	}

	err := objects.Set("nonexistent", "value")
	require.Error(t, err)

	// Should contain errors for both objects
	assert.Contains(t, err.Error(), "object 0")
	assert.Contains(t, err.Error(), "object 1")
	assert.Contains(t, err.Error(), "does not exist")
}

func TestServerObjectsSetPartialErrors(t *testing.T) {
	objects := ServerObjects{
		{
			attributes: map[string]any{"hostname": "server1", "memory": 16, "object_id": float64(1)},
			oldValues:  map[string]any{},
		},
		{
			attributes: map[string]any{"hostname": "server2", "object_id": float64(2)},
			oldValues:  map[string]any{},
		},
	}

	// "memory" exists in first object but not second
	err := objects.Set("memory", 32)
	require.Error(t, err)

	// First object should be updated successfully
	assert.Equal(t, 32, objects[0].Get("memory"))
	assert.Equal(t, 16, objects[0].oldValues["memory"])

	// Error should only mention the second object
	assert.Contains(t, err.Error(), "object 1")
	assert.Contains(t, err.Error(), "id=2")
	assert.NotContains(t, err.Error(), "object 0")
}

func TestServerObjectsSetEmpty(t *testing.T) {
	objects := ServerObjects{}
	err := objects.Set("hostname", "value")
	require.NoError(t, err) // No objects = no errors
}

func TestServerObjectsDelete(t *testing.T) {
	objects := ServerObjects{
		{
			attributes: map[string]any{"hostname": "server1", "object_id": float64(1)},
			oldValues:  map[string]any{},
		},
		{
			attributes: map[string]any{"hostname": "server2", "object_id": float64(2)},
			oldValues:  map[string]any{},
		},
	}

	objects.Delete()

	assert.True(t, objects[0].deleted)
	assert.True(t, objects[1].deleted)
	assert.Equal(t, "deleted", objects[0].CommitState())
	assert.Equal(t, "deleted", objects[1].CommitState())
}

func TestServerObjectsDeleteEmpty(_ *testing.T) {
	objects := ServerObjects{}
	objects.Delete() // Should not panic
}

func TestServerObjectsSetWithCommit(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(200)
		w.Write([]byte(`{"status": "success", "commit_id": 999}`))
	}))
	defer server.Close()

	resetConfig()
	t.Setenv("SERVERADMIN_TOKEN", "testtoken")
	t.Setenv("SERVERADMIN_BASE_URL", server.URL)

	objects := ServerObjects{
		{
			attributes: map[string]any{"hostname": "server1", "object_id": float64(1)},
			oldValues:  map[string]any{},
		},
		{
			attributes: map[string]any{"hostname": "server2", "object_id": float64(2)},
			oldValues:  map[string]any{},
		},
	}

	// Set valid attribute
	err := objects.Set("hostname", "updated.local")
	require.NoError(t, err)

	// Commit should work
	commitID, err := objects.Commit()
	require.NoError(t, err)
	assert.Equal(t, 999, commitID)

	// State should be consistent after commit
	assert.Equal(t, "consistent", objects[0].CommitState())
	assert.Equal(t, "consistent", objects[1].CommitState())
}
