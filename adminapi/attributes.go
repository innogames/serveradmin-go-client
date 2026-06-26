package adminapi

import (
	"context"
	"encoding/json"
	"fmt"
)

const apiEndpointAttributes = "/api/dataset/attributes"

// Attribute describes a single attribute definition as returned by the
// Serveradmin dataset/attributes endpoint. It covers both regular attributes
// stored in the attribute table and the special attributes (e.g. hostname,
// servertype) that are not stored there but remain queryable like any other
// attribute.
type Attribute struct {
	// AttributeID is the unique identifier (and name) of the attribute.
	AttributeID string `json:"attribute_id"`
	// Type is the attribute's data type (e.g. "string", "boolean", "relation", "inet").
	Type string `json:"type"`
	// Multi reports whether the attribute holds multiple values.
	Multi bool `json:"multi"`
	// Hovertext is the human-readable description of the attribute.
	Hovertext string `json:"hovertext"`
	// Group is the grouping label the attribute belongs to.
	Group string `json:"group"`
	// HelpLink is an optional link to documentation for the attribute.
	HelpLink string `json:"help_link"`
	// InetAddressFamily is the network address family for inet-typed attributes.
	InetAddressFamily string `json:"inet_address_family"`
	// Readonly reports whether the attribute can be modified.
	Readonly bool `json:"readonly"`
	// Clone reports whether the attribute's value is copied when cloning an object.
	Clone bool `json:"clone"`
	// History reports whether changes to the attribute are tracked in history.
	History bool `json:"history"`
	// Regexp is an optional validation pattern for the attribute's values.
	Regexp string `json:"regexp"`
	// ReversedAttribute is the attribute_id this attribute is the reverse of,
	// or empty if it is not a reversed relation.
	ReversedAttribute string `json:"reversed_attribute"`
	// TargetServertypes lists the servertype IDs this attribute is attached to.
	// It is empty for special attributes, which are not stored in the database.
	TargetServertypes []string `json:"target_servertypes"`
}

// attributesResponse mirrors {"status": "success", "result": [{...}, ...]}
type attributesResponse struct {
	Status  string      `json:"status"`
	Result  []Attribute `json:"result"`
	Message string      `json:"message"`
}

// FetchAttributes retrieves all attribute definitions from the Serveradmin
// server using this client. The result includes the special attributes (e.g.
// hostname, servertype) that are not stored in the attribute table but remain
// queryable, and is suitable for auto-completion or attribute selection.
func (c *Client) FetchAttributes(ctx context.Context) ([]Attribute, error) {
	// The endpoint takes no input; send an empty JSON object so the request
	// body is valid for the API's signature verification.
	resp, err := c.sendRequest(ctx, apiEndpointAttributes, struct{}{})
	if err != nil {
		return nil, fmt.Errorf("fetching %s: %w", apiEndpointAttributes, err)
	}
	defer resp.Body.Close()

	var result attributesResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decoding attributes response: %w", err)
	}

	if result.Status == "error" {
		return nil, fmt.Errorf("fetching attributes failed: %s", result.Message)
	}

	return result.Result, nil
}
