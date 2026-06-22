package adminapi

import (
	"context"
	"encoding/json"
	"fmt"
	"slices"
)

// Query is a struct to build a query to the SA API
type Query struct {
	client               *Client
	filters              Filters
	restrictedAttributes []string
	orderBy              string
	loaded               bool
	serverObjects        ServerObjects
}

// Attributes is a map of attributes, indexed by attribute name
type Attributes map[string]any

// Has checks if the given key exists in the attributes map
func (a Attributes) Has(key string) bool {
	_, ok := a[key]
	return ok
}

// FromQuery creates a new Query object from a query string.
//
// Deprecated: use Client.FromQuery so the request uses an explicit, per-instance
// configuration instead of a process-global one built from environment variables.
func FromQuery(query string) (Query, error) {
	return newQueryFromString(nil, query)
}

// NewQuery initializes a new query which loads data from SA if needed.
//
// Deprecated: use Client.NewQuery so the request uses an explicit, per-instance
// configuration instead of a process-global one built from environment variables.
func NewQuery(filters Filters) Query {
	return newQuery(nil, filters)
}

// FromQuery creates a new Query object from a query string, bound to this client.
func (c *Client) FromQuery(query string) (Query, error) {
	return newQueryFromString(c, query)
}

// NewQuery initializes a new query bound to this client.
func (c *Client) NewQuery(filters Filters) Query {
	return newQuery(c, filters)
}

func newQuery(client *Client, filters Filters) Query {
	return Query{
		client:               client,
		filters:              filters,
		restrictedAttributes: []string{"object_id", "hostname"},
	}
}

func newQueryFromString(client *Client, query string) (Query, error) {
	filters, err := ParseQuery(query)
	if err != nil {
		return Query{}, fmt.Errorf("parsing query %s: %w", query, err)
	}

	return newQuery(client, filters), nil
}

// SetAttributes replaces the list of attributes to fetch from the API
func (q *Query) SetAttributes(attributes ...string) {
	q.restrictedAttributes = attributes
}

// AddAttributes appends additional attributes to the list of attributes to fetch
func (q *Query) AddAttributes(attributes ...string) {
	q.restrictedAttributes = append(q.restrictedAttributes, attributes...)
}

// OrderBy sets the attribute to sort results by
func (q *Query) OrderBy(attribute string) {
	q.orderBy = attribute
}

// AddFilter adds or updates a filter for the specified attribute
func (q *Query) AddFilter(attribute string, filter any) {
	q.filters[attribute] = filter
}

// Count matching SA objects
func (q *Query) Count(ctx context.Context) (int, error) {
	err := q.load(ctx)
	if err != nil {
		return 0, err
	}

	return len(q.serverObjects), nil
}

// All returns all matching SA objects
func (q *Query) All(ctx context.Context) (ServerObjects, error) {
	err := q.load(ctx)
	if err != nil {
		return nil, err
	}

	return q.serverObjects, nil
}

// One returns exactly one matching SA object. If there is none or more than one, an error is returned.
// Returns ErrNoResults if no objects match, or a wrapped ErrMultipleResults if more than one matches.
func (q *Query) One(ctx context.Context) (*ServerObject, error) {
	err := q.load(ctx)
	if err != nil {
		return nil, err
	}

	switch len(q.serverObjects) {
	case 1:
		return q.serverObjects[0], nil
	case 0:
		return nil, ErrNoResults
	default:
		return nil, fmt.Errorf("got %d: %w", len(q.serverObjects), ErrMultipleResults)
	}
}

func (q *Query) load(ctx context.Context) error {
	if q.loaded {
		return nil
	}

	client, err := q.resolveClient()
	if err != nil {
		return err
	}

	// always add "object_id" as attribute as we need it to modify the object
	if !slices.Contains(q.restrictedAttributes, "object_id") {
		q.restrictedAttributes = append(q.restrictedAttributes, "object_id")
	}

	request := queryRequest{
		Filters:    q.filters,
		Restricted: q.restrictedAttributes,
		OrderBy:    q.orderBy, // todo fix serverside ordering in API or do it on client side
	}

	resp, err := client.sendRequest(ctx, apiEndpointQuery, request)
	if err != nil {
		return fmt.Errorf("querying %s: %w", apiEndpointQuery, err)
	}
	defer resp.Body.Close()

	respServer := queryResponse{}
	if err = json.NewDecoder(resp.Body).Decode(&respServer); err != nil {
		return fmt.Errorf("decoding query response: %w", err)
	}

	// map attribute map into ServerObject objects, stamping the client so later
	// Commit calls reuse the same configuration.
	q.serverObjects = make(ServerObjects, len(respServer.Result))
	for idx, object := range respServer.Result {
		q.serverObjects[idx] = &ServerObject{
			client:     client,
			attributes: object,
			oldValues:  Attributes{},
		}
	}
	q.loaded = true

	return nil
}

// resolveClient returns the query's bound client, falling back to the lazily
// built environment-based default client for the deprecated package-level API.
func (q *Query) resolveClient() (*Client, error) {
	if q.client != nil {
		return q.client, nil
	}
	return defaultClient()
}

// like {"Filters": {"hostname": {"Regexp": "foo.local.*"}}, "restrict": ["hostname", "object_id"]}
type queryRequest struct {
	Filters    map[string]any `json:"filters"`
	Restricted []string       `json:"restrict"`
	OrderBy    string         `json:"order_by,omitempty"`
}

// like {"status": "success", "result": [{"object_id": 483903, "hostname": "foo.local"}]}
type queryResponse struct {
	Status string       `json:"status"`
	Result []Attributes `json:"result"`
}
