package adminapi

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFetchAttributesSuccess(t *testing.T) {
	var requestPath string
	var requestBody string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestPath = r.URL.Path
		body, _ := io.ReadAll(r.Body)
		requestBody = string(body)

		w.WriteHeader(200)
		_, _ = w.Write([]byte(`{
			"status": "success",
			"result": [
				{
					"attribute_id": "hostname",
					"type": "string",
					"multi": false,
					"hovertext": "The hostname",
					"group": "base",
					"help_link": "",
					"inet_address_family": "",
					"readonly": true,
					"clone": false,
					"history": true,
					"regexp": null,
					"reversed_attribute": null,
					"target_servertypes": []
				},
				{
					"attribute_id": "responsible_admins",
					"type": "relation",
					"multi": true,
					"hovertext": "Admins responsible for the object",
					"group": "base",
					"help_link": "https://example.com/help",
					"inet_address_family": "",
					"readonly": false,
					"clone": true,
					"history": true,
					"regexp": "^[a-z]+$",
					"reversed_attribute": "responsible_for",
					"target_servertypes": ["vm", "hardware"]
				}
			]
		}`))
	}))
	defer server.Close()

	client := mustClient(t, server.URL)

	attributes, err := client.FetchAttributes(context.Background())
	require.NoError(t, err)
	require.Len(t, attributes, 2)

	// Request hits the dataset/attributes endpoint with a valid JSON body.
	assert.Equal(t, "/api/dataset/attributes", requestPath)
	assert.Equal(t, "{}", requestBody)

	// Special attribute with null regexp/reversed_attribute and no target servertypes.
	hostname := attributes[0]
	assert.Equal(t, "hostname", hostname.AttributeID)
	assert.Equal(t, "string", hostname.Type)
	assert.False(t, hostname.Multi)
	assert.Equal(t, "The hostname", hostname.Hovertext)
	assert.Equal(t, "base", hostname.Group)
	assert.True(t, hostname.Readonly)
	assert.True(t, hostname.History)
	assert.Empty(t, hostname.Regexp)
	assert.Empty(t, hostname.ReversedAttribute)
	assert.Empty(t, hostname.TargetServertypes)

	// Regular multi attribute with all optional fields populated.
	admins := attributes[1]
	assert.Equal(t, "responsible_admins", admins.AttributeID)
	assert.Equal(t, "relation", admins.Type)
	assert.True(t, admins.Multi)
	assert.True(t, admins.Clone)
	assert.Equal(t, "https://example.com/help", admins.HelpLink)
	assert.Equal(t, "^[a-z]+$", admins.Regexp)
	assert.Equal(t, "responsible_for", admins.ReversedAttribute)
	assert.Equal(t, []string{"vm", "hardware"}, admins.TargetServertypes)
}

func TestFetchAttributesEmpty(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(200)
		_, _ = w.Write([]byte(`{"status": "success", "result": []}`))
	}))
	defer server.Close()

	client := mustClient(t, server.URL)

	attributes, err := client.FetchAttributes(context.Background())
	require.NoError(t, err)
	assert.Empty(t, attributes)
}

func TestFetchAttributesError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(200)
		_, _ = w.Write([]byte(`{"status": "error", "message": "something went wrong"}`))
	}))
	defer server.Close()

	client := mustClient(t, server.URL)

	attributes, err := client.FetchAttributes(context.Background())
	require.Error(t, err)
	assert.Nil(t, attributes)
	assert.Contains(t, err.Error(), "something went wrong")
}

func TestFetchAttributesHTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(403)
		_, _ = w.Write([]byte(`{"error": {"message": "Forbidden: No known public key found"}}`))
	}))
	defer server.Close()

	client := mustClient(t, server.URL)

	attributes, err := client.FetchAttributes(context.Background())
	require.Error(t, err)
	assert.Nil(t, attributes)
	assert.Contains(t, err.Error(), "HTTP error 403 Forbidden")
}
