package adminapi

import (
	"encoding/json"
	"fmt"
	"slices"
)

// Query is a struct to build a query to the SA API
type Query struct {
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

// FromQuery creates a new Query object from a query string
func FromQuery(query string) (Query, error) {
	filters, err := ParseQuery(query)
	if err != nil {
		return Query{}, fmt.Errorf("parsing query %s: %w", query, err)
	}

	return NewQuery(filters), nil
}

// NewQuery initialize a new query which loads data from SA if needed
func NewQuery(filters Filters) Query {
	return Query{
		filters:              filters,
		restrictedAttributes: []string{"object_id", "hostname"},
	}
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
func (q *Query) Count() (int, error) {
	err := q.load()
	if err != nil {
		return 0, err
	}

	return len(q.serverObjects), nil
}

// All returns all matching SA objects
func (q *Query) All() (ServerObjects, error) {
	err := q.load()
	if err != nil {
		return nil, err
	}

	return q.serverObjects, nil
}

// One returns exactly one matching SA object. If there is none or more than one, an error is returned.
// Returns ErrNoResults if no objects match, or a wrapped ErrMultipleResults if more than one matches.
func (q *Query) One() (*ServerObject, error) {
	err := q.load()
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

func (q *Query) load() error {
	if q.loaded {
		return nil
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

	resp, err := sendRequest(apiEndpointQuery, request)
	if err != nil {
		return fmt.Errorf("querying %s: %w", apiEndpointQuery, err)
	}
	defer resp.Body.Close()

	respServer := queryResponse{}
	if err = json.NewDecoder(resp.Body).Decode(&respServer); err != nil {
		return fmt.Errorf("decoding query response: %w", err)
	}

	// map attribute map into ServerObject objects
	q.serverObjects = make(ServerObjects, len(respServer.Result))
	for idx, object := range respServer.Result {
		q.serverObjects[idx] = &ServerObject{
			attributes: object,
			oldValues:  Attributes{},
		}
	}
	q.loaded = true

	return nil
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
