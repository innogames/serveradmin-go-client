package adminapi

import (
	"context"
	"encoding/json"
	"fmt"
)

const apiEndpointCall = "/call"

type callRequest struct {
	Group  string         `json:"group"`
	Name   string         `json:"name"`
	Args   []any          `json:"args"`
	Kwargs map[string]any `json:"kwargs"`
}

type callResponse struct {
	Status  string `json:"status"`
	RetVal  any    `json:"retval"`
	Message string `json:"message"`
}

// CallAPI calls a remote API function on the Serveradmin server.
// It takes a function group, function name, and keyword arguments as a map.
//
// Deprecated: use Client.CallAPI so the request uses an explicit, per-instance
// configuration instead of a process-global one built from environment variables.
func CallAPI(group, function string, args map[string]any) (any, error) {
	client, err := defaultClient()
	if err != nil {
		return nil, err
	}
	return client.CallAPI(context.Background(), group, function, args)
}

// CallAPI calls a remote API function on the Serveradmin server using this client.
// It takes a function group, function name, and keyword arguments as a map.
func (c *Client) CallAPI(ctx context.Context, group, function string, args map[string]any) (any, error) {
	req := callRequest{
		Group:  group,
		Name:   function,
		Args:   []any{},
		Kwargs: args,
	}

	resp, err := c.sendRequest(ctx, apiEndpointCall, req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result callResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode call response: %w", err)
	}

	if result.Status == "error" {
		return nil, fmt.Errorf("API call %s.%s failed: %s", group, function, result.Message)
	}

	return result.RetVal, nil
}
