package adminapi

import (
	"encoding/json"
	"fmt"
	"reflect"
)

// ServerObjects is a slice of ServerObject pointers
type ServerObjects []*ServerObject

// ServerObject is a map of key-value attributes of a SA object
type ServerObject struct {
	attributes Attributes
	oldValues  Attributes // tracks original values before first modification
	deleted    bool
}

// Get safely retrieves an attribute, converting JSON float64 numbers to int when needed
func (s *ServerObject) Get(attribute string) any {
	if val, ok := s.attributes[attribute]; ok {
		if floatVal, isFloat := val.(float64); isFloat {
			return int(floatVal)
		}
		return val
	}
	return nil
}

// GetString safely retrieves an attribute as a string
func (s *ServerObject) GetString(attribute string) string {
	val := s.Get(attribute)
	if strVal, isString := val.(string); isString {
		return strVal
	}
	return ""
}

// GetMulti safely retrieves a multi-valued attribute as a MultiAttr.
// Returns an empty MultiAttr if the attribute is missing, nil, or not a slice of strings.
func (s *ServerObject) GetMulti(attribute string) MultiAttr {
	val, ok := s.attributes[attribute]
	if !ok || val == nil {
		return MultiAttr{}
	}

	switch v := val.(type) {
	case MultiAttr:
		return v
	case []string:
		return v
	case []any:
		result := make(MultiAttr, 0, len(v))
		for _, elem := range v {
			if str, ok := elem.(string); ok {
				result = append(result, str)
			}
		}
		return result
	default:
		return MultiAttr{}
	}
}

// ObjectID returns the "object_id" attribute of the ServerObject
func (s *ServerObject) ObjectID() int {
	val := s.Get("object_id")
	if id, ok := val.(int); ok {
		return id
	}
	return 0
}

// CommitState represents the state of a ServerObject with respect to pending changes.
type CommitState string

const (
	// StateCreated indicates the object is new and has not been committed yet.
	StateCreated CommitState = "created"
	// StateDeleted indicates the object has been marked for deletion.
	StateDeleted CommitState = "deleted"
	// StateChanged indicates the object has local modifications pending commit.
	StateChanged CommitState = "changed"
	// StateConsistent indicates the object has no pending changes.
	StateConsistent CommitState = "consistent"
)

// Set modifies an attribute value and tracks the change for commit.
func (s *ServerObject) Set(key string, value any) error {
	if _, exists := s.attributes[key]; !exists {
		return fmt.Errorf("attribute %q: %w", key, ErrUnknownAttribute)
	}

	// Save the original value on first modification only
	if _, tracked := s.oldValues[key]; !tracked {
		old := s.attributes[key]
		// Deep copy slices to prevent aliasing (handle any slice type)
		if oldSlice := toAnySlice(old); oldSlice != nil {
			cp := make([]any, len(oldSlice))
			copy(cp, oldSlice)
			s.oldValues[key] = cp
		} else {
			s.oldValues[key] = old
		}
	}

	s.attributes[key] = value
	return nil
}

// Delete marks the object for deletion on the next commit.
func (s *ServerObject) Delete() {
	s.deleted = true
}

// Rollback reverts all local changes, restoring original attribute values.
func (s *ServerObject) Rollback() {
	s.deleted = false
	for key, oldVal := range s.oldValues {
		s.attributes[key] = oldVal
	}
	s.oldValues = Attributes{}
}

// CommitState returns the current state of the object with respect to pending changes.
func (s *ServerObject) CommitState() CommitState {
	if s.attributes["object_id"] == nil {
		return StateCreated
	}
	if s.deleted {
		return StateDeleted
	}
	for key, oldVal := range s.oldValues {
		newVal := s.attributes[key]
		if !jsonEqual(oldVal, newVal) {
			return StateChanged
		}
	}
	return StateConsistent
}

// serializeChanges builds the change delta for commit payload.
func (s *ServerObject) serializeChanges() Attributes {
	changes := Attributes{"object_id": s.ObjectID()}

	for key, oldVal := range s.oldValues {
		newVal := s.attributes[key]
		if jsonEqual(oldVal, newVal) {
			continue
		}

		// Check if both old and new values are slices (of any type)
		oldSlice := toAnySlice(oldVal)
		newSlice := toAnySlice(newVal)

		if oldSlice != nil && newSlice != nil {
			// Multi-attribute: compute add/remove sets
			add, remove := sliceDiff(oldSlice, newSlice)
			changes[key] = map[string]any{
				"action": "multi",
				"add":    add,
				"remove": remove,
			}
		} else {
			changes[key] = map[string]any{
				"action": "update",
				"old":    oldVal,
				"new":    newVal,
			}
		}
	}

	return changes
}

func (s *ServerObject) confirmChanges() {
	s.oldValues = Attributes{}
	if s.deleted {
		s.attributes["object_id"] = nil
		s.deleted = false
	}
}

// jsonEqual compares two values using JSON serialization for consistency with the Python client.
func jsonEqual(a, b any) bool {
	aj, _ := json.Marshal(a)
	bj, _ := json.Marshal(b)
	return string(aj) == string(bj)
}

// toAnySlice converts any slice type ([]string, []int, []any, etc.) to []any.
// Returns nil if v is not a slice.
func toAnySlice(v any) []any {
	if v == nil {
		return nil
	}

	// Fast path for []any
	if s, ok := v.([]any); ok {
		return s
	}

	// Use reflection for other slice types
	rv := reflect.ValueOf(v)
	if rv.Kind() != reflect.Slice {
		return nil
	}

	result := make([]any, rv.Len())
	for i := range rv.Len() {
		result[i] = rv.Index(i).Interface()
	}
	return result
}

// sliceDiff computes elements added to and removed from old to produce new (set semantics).
func sliceDiff(old, cur []any) (add, remove []any) {
	// Initialize as empty slices instead of nil so JSON serializes to [] not null
	add = []any{}
	remove = []any{}

	oldSet := make(map[string]any, len(old))
	for _, v := range old {
		k, _ := json.Marshal(v)
		oldSet[string(k)] = v
	}
	curSet := make(map[string]any, len(cur))
	for _, v := range cur {
		k, _ := json.Marshal(v)
		curSet[string(k)] = v
	}

	for k, v := range curSet {
		if _, exists := oldSet[k]; !exists {
			add = append(add, v)
		}
	}
	for k, v := range oldSet {
		if _, exists := curSet[k]; !exists {
			remove = append(remove, v)
		}
	}
	return add, remove
}
