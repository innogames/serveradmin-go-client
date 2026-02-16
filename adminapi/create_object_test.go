package adminapi

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewObject(t *testing.T) {
	tests := []struct {
		name          string
		serverType    string
		attributes    Attributes
		newObjResp    string
		commitResp    string
		queryResp     string
		expectError   bool
		expectedErrIs string
	}{
		{
			name:       "successful object creation",
			serverType: "vm",
			attributes: Attributes{
				"hostname":    "new-server.local",
				"environment": "development",
			},
			newObjResp: `{
				"status": "success",
				"result": {
					"hostname": "",
					"environment": "",
					"num_cpu": 4,
					"memory": 8192,
					"servertype": "vm"
				}
			}`,
			commitResp: `{"status": "success", "commit_id": 42}`,
			queryResp: `{
				"status": "success",
				"result": [{
					"object_id": 12345,
					"hostname": "new-server.local",
					"environment": "development"
				}]
			}`,
		},
		{
			name:       "minimal attributes",
			serverType: "loadbalancer",
			attributes: Attributes{
				"hostname": "lb-1.local",
			},
			newObjResp: `{
				"status": "success",
				"result": {
					"hostname": "",
					"servertype": "loadbalancer"
				}
			}`,
			commitResp: `{"status": "success", "commit_id": 43}`,
			queryResp: `{
				"status": "success",
				"result": [{
					"object_id": 99,
					"hostname": "lb-1.local"
				}]
			}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			callCount := 0
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				callCount++
				switch r.URL.Path {
				case "/api/dataset/new_object":
					assert.Equal(t, tt.serverType, r.URL.Query().Get("servertype"))
					w.WriteHeader(http.StatusOK)
					_, _ = w.Write([]byte(tt.newObjResp))
				case "/api/dataset/commit":
					w.WriteHeader(http.StatusOK)
					_, _ = w.Write([]byte(tt.commitResp))
				case "/api/dataset/query":
					w.WriteHeader(http.StatusOK)
					_, _ = w.Write([]byte(tt.queryResp))
				default:
					t.Fatalf("unexpected request to %s", r.URL.Path)
				}
			}))
			defer server.Close()

			resetConfig()
			t.Setenv("SERVERADMIN_TOKEN", "test-token-1234")
			t.Setenv("SERVERADMIN_BASE_URL", server.URL)

			obj, err := NewObject(tt.serverType, tt.attributes)

			require.NoError(t, err)
			require.NotNil(t, obj)

			// Verify object_id is set from re-query
			assert.NotNil(t, obj.Get("object_id"), "object_id should be set after creation")
			assert.Positive(t, obj.ObjectID(), "object_id should be positive")

			// Verify hostname is present
			assert.Equal(t, tt.attributes["hostname"], obj.GetString("hostname"))

			// Verify state is consistent (committed)
			assert.Equal(t, StateConsistent, obj.CommitState())

			// Verify all 3 API calls were made
			assert.Equal(t, 3, callCount, "should make new_object, commit, and query calls")
		})
	}
}

func TestNewObject_MissingHostname(t *testing.T) {
	obj, err := NewObject("vm", Attributes{"environment": "dev"})

	require.Error(t, err)
	assert.Nil(t, obj)
	require.ErrorIs(t, err, ErrUnknownAttribute)
	assert.Contains(t, err.Error(), "hostname")
}

func TestNewObject_UnknownAttribute(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{
			"status": "success",
			"result": {
				"hostname": "",
				"servertype": "vm"
			}
		}`))
	}))
	defer server.Close()

	resetConfig()
	t.Setenv("SERVERADMIN_TOKEN", "test-token-1234")
	t.Setenv("SERVERADMIN_BASE_URL", server.URL)

	obj, err := NewObject("vm", Attributes{
		"hostname":          "test.local",
		"nonexistent_field": "value",
	})

	require.Error(t, err)
	assert.Nil(t, obj)
	assert.Contains(t, err.Error(), "nonexistent_field")
}

func TestNewObject_HTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error": {"message": "Bad Request: Invalid servertype"}}`))
	}))
	defer server.Close()

	resetConfig()
	t.Setenv("SERVERADMIN_TOKEN", "test-token-1234")
	t.Setenv("SERVERADMIN_BASE_URL", server.URL)

	obj, err := NewObject("invalid-type", Attributes{"hostname": "test.local"})

	require.Error(t, err)
	assert.Nil(t, obj)
	assert.Contains(t, err.Error(), "HTTP error 400 Bad Request")
}

func TestNewObject_CommitFailure(t *testing.T) {
	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		switch r.URL.Path {
		case "/api/dataset/new_object":
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{
				"status": "success",
				"result": {"hostname": "", "servertype": "vm"}
			}`))
		case "/api/dataset/commit":
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"status": "error", "message": "validation failed"}`))
		default:
			t.Fatalf("unexpected request to %s", r.URL.Path)
		}
	}))
	defer server.Close()

	resetConfig()
	t.Setenv("SERVERADMIN_TOKEN", "test-token-1234")
	t.Setenv("SERVERADMIN_BASE_URL", server.URL)

	obj, err := NewObject("vm", Attributes{"hostname": "test.local"})

	require.Error(t, err)
	assert.Nil(t, obj)
	assert.Contains(t, err.Error(), "committing new object")

	// Should not have made the query call
	assert.Equal(t, 2, callCount, "should only make new_object and commit calls")
}

func TestNewObject_CommitPayload(t *testing.T) {
	var receivedCommit commitRequest
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/dataset/new_object":
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{
				"status": "success",
				"result": {"hostname": "", "servertype": "vm", "project": ""}
			}`))
		case "/api/dataset/commit":
			_ = json.NewDecoder(r.Body).Decode(&receivedCommit)
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"status": "success", "commit_id": 1}`))
		case "/api/dataset/query":
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{
				"status": "success",
				"result": [{"object_id": 1, "hostname": "test.local", "project": "admin"}]
			}`))
		}
	}))
	defer server.Close()

	resetConfig()
	t.Setenv("SERVERADMIN_TOKEN", "test-token-1234")
	t.Setenv("SERVERADMIN_BASE_URL", server.URL)

	_, err := NewObject("vm", Attributes{
		"hostname": "test.local",
		"project":  "admin",
	})
	require.NoError(t, err)

	// Verify the commit payload contains a created object
	require.Len(t, receivedCommit.Created, 1)
	assert.Equal(t, "test.local", receivedCommit.Created[0]["hostname"])
	assert.Equal(t, "admin", receivedCommit.Created[0]["project"])
}
