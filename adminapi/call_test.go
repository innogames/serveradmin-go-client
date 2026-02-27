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

func TestCallAPISuccess(t *testing.T) {
	var receivedBody callRequest

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		json.Unmarshal(body, &receivedBody)

		w.WriteHeader(200)
		w.Write([]byte(`{"status": "success", "retval": "10.0.0.1"}`))
	}))
	defer server.Close()

	resetConfig()
	t.Setenv("SERVERADMIN_TOKEN", "testtoken")
	t.Setenv("SERVERADMIN_BASE_URL", server.URL)

	result, err := CallAPI("ip", "get_free", map[string]any{"network": "internal"})
	require.NoError(t, err)
	assert.Equal(t, "10.0.0.1", result)

	// Verify request structure
	assert.Equal(t, "ip", receivedBody.Group)
	assert.Equal(t, "get_free", receivedBody.Name)
	assert.Empty(t, receivedBody.Args)
	assert.Equal(t, map[string]any{"network": "internal"}, receivedBody.Kwargs)
}

func TestCallAPIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(200)
		w.Write([]byte(`{"status": "error", "message": "function not found"}`))
	}))
	defer server.Close()

	resetConfig()
	t.Setenv("SERVERADMIN_TOKEN", "testtoken")
	t.Setenv("SERVERADMIN_BASE_URL", server.URL)

	result, err := CallAPI("ip", "nonexistent", map[string]any{})
	assert.Nil(t, result)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "ip.nonexistent")
	assert.Contains(t, err.Error(), "function not found")
}

func TestCallAPIComplexReturnValue(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(200)
		w.Write([]byte(`{"status": "success", "retval": {"ip": "10.0.0.1", "network": "internal"}}`))
	}))
	defer server.Close()

	resetConfig()
	t.Setenv("SERVERADMIN_TOKEN", "testtoken")
	t.Setenv("SERVERADMIN_BASE_URL", server.URL)

	result, err := CallAPI("ip", "get_details", map[string]any{"ip": "10.0.0.1"})
	require.NoError(t, err)

	resultMap, ok := result.(map[string]any)
	require.True(t, ok, "expected map return value")
	assert.Equal(t, "10.0.0.1", resultMap["ip"])
	assert.Equal(t, "internal", resultMap["network"])
}

func TestCallAPINilArgs(t *testing.T) {
	var receivedBody callRequest

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		json.Unmarshal(body, &receivedBody)

		w.WriteHeader(200)
		w.Write([]byte(`{"status": "success", "retval": null}`))
	}))
	defer server.Close()

	resetConfig()
	t.Setenv("SERVERADMIN_TOKEN", "testtoken")
	t.Setenv("SERVERADMIN_BASE_URL", server.URL)

	result, err := CallAPI("system", "ping", nil)
	require.NoError(t, err)
	assert.Nil(t, result)

	assert.Equal(t, "system", receivedBody.Group)
	assert.Equal(t, "ping", receivedBody.Name)
}
