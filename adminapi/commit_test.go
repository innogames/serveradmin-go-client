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
