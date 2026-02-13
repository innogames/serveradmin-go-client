package adminapi

import (
	"net/http"
	"net/http/httptest"
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

func TestNewObject(t *testing.T) {
	tests := []struct {
		name             string
		serverType       string
		mockResponse     string
		expectedAttrs    map[string]any
		expectError      bool
		expectedErrorMsg string
	}{
		{
			name:       "successful object creation",
			serverType: "vm",
			mockResponse: `{
				"status": "success",
				"result": {
					"hostname": "new-server.local",
					"environment": "development",
					"num_cpu": 4,
					"memory": 8192,
					"servertype": "vm"
				}
			}`,
			expectedAttrs: map[string]any{
				"hostname":    "new-server.local",
				"environment": "development",
				"num_cpu":     float64(4),
				"memory":      float64(8192),
				"servertype":  "vm",
				"object_id":   nil,
			},
			expectError: false,
		},
		{
			name:       "minimal object attributes",
			serverType: "loadbalancer",
			mockResponse: `{
				"status": "success",
				"result": {
					"servertype": "loadbalancer"
				}
			}`,
			expectedAttrs: map[string]any{
				"servertype": "loadbalancer",
				"object_id":  nil,
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mock server
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// Verify request path and query parameters
				assert.Equal(t, "/api/dataset/new_object", r.URL.Path)
				assert.Equal(t, tt.serverType, r.URL.Query().Get("servertype"))
				assert.Equal(t, "GET", r.Method)

				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte(tt.mockResponse))
			}))
			defer server.Close()

			// Configure test environment
			resetConfig()
			t.Setenv("SERVERADMIN_TOKEN", "test-token-1234")
			t.Setenv("SERVERADMIN_BASE_URL", server.URL)

			// Call NewObject
			obj, err := NewObject(tt.serverType)

			if tt.expectError {
				require.Error(t, err)
				if tt.expectedErrorMsg != "" {
					assert.Contains(t, err.Error(), tt.expectedErrorMsg)
				}
				return
			}

			// Verify no error
			require.NoError(t, err)
			require.NotNil(t, obj)

			// Verify attributes
			assert.Equal(t, tt.expectedAttrs, obj.attributes,
				"attributes should match expected values")

			// Verify object_id is nil (new object)
			assert.Nil(t, obj.Get("object_id"),
				"object_id should be nil for new objects")

			// Verify commit state is "created"
			assert.Equal(t, StateCreated, obj.CommitState(),
				"new object should have commit state 'created'")

			// Verify oldValues is empty
			assert.Empty(t, obj.oldValues,
				"new object should have empty oldValues")

			// Verify object is not marked as deleted
			assert.False(t, obj.deleted,
				"new object should not be marked as deleted")
		})
	}
}

func TestNewObject_HTTPError(t *testing.T) {
	// Create mock server that returns an error
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error": {"message": "Bad Request: Invalid servertype"}}`))
	}))
	defer server.Close()

	// Configure test environment
	resetConfig()
	t.Setenv("SERVERADMIN_TOKEN", "test-token-1234")
	t.Setenv("SERVERADMIN_BASE_URL", server.URL)

	// Call NewObject
	obj, err := NewObject("invalid-type")

	// Verify error is returned
	require.Error(t, err)
	assert.Nil(t, obj)
	assert.Contains(t, err.Error(), "HTTP error 400 Bad Request")
}
