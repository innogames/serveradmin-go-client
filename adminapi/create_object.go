package adminapi

import (
	"encoding/json"
	"fmt"
	"net/url"
)

// NewObject creates a new server object with the given attributes, commits it,
// and returns the fully populated object with a server-assigned object_id.
// The attributes map must include "hostname".
func NewObject(serverType string, attributes Attributes) (*ServerObject, error) {
	if !attributes.Has("hostname") {
		return nil, fmt.Errorf("attributes must include %q: %w", "hostname", ErrUnknownAttribute)
	}

	server := &ServerObject{
		oldValues: Attributes{},
	}

	// Fetch default attributes from the API
	params := url.Values{}
	params.Add("servertype", serverType)
	fullURL := apiEndpointNewObject + "?" + params.Encode()

	resp, err := sendRequest(fullURL, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var response struct {
		Result Attributes `json:"result"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, err
	}
	server.attributes = response.Result

	// Ensure object_id is nil so CommitState() returns "created"
	server.attributes["object_id"] = nil

	// Apply caller-provided attributes (validates each exists in schema)
	for key, value := range attributes {
		if err := server.Set(key, value); err != nil {
			return nil, fmt.Errorf("setting attribute %q: %w", key, err)
		}
	}

	// Commit the new object
	if _, err := server.Commit(); err != nil {
		return nil, fmt.Errorf("committing new object: %w", err)
	}

	// Re-query to get the server-assigned object_id
	q := NewQuery(Filters{"hostname": attributes["hostname"]})
	created, err := q.One()
	if err != nil {
		return nil, fmt.Errorf("re-querying created object: %w", err)
	}
	_ = server.Set("object_id", created.ObjectID())

	return created, nil
}
