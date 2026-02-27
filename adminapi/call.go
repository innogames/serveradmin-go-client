package adminapi

import (
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
func CallAPI(group, function string, args map[string]any) (any, error) {
	req := callRequest{
		Group:  group,
		Name:   function,
		Args:   []any{},
		Kwargs: args,
	}

	resp, err := sendRequest(apiEndpointCall, req)
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
