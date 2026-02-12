package adminapi

import (
	"encoding/json"
	"fmt"
)

// ServerObjects is a slice of ServerObject pointers
type ServerObjects []*ServerObject

// ServerObject is a map of key-value attributes of a SA object
type ServerObject struct {
	attributes map[string]any
	oldValues  map[string]any // tracks original values before first modification
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

// ObjectID returns the "object_id" attribute of the ServerObject
func (s *ServerObject) ObjectID() int {
	val := s.Get("object_id")
	if id, ok := val.(int); ok {
		return id
	}
	return 0
}

// Set modifies an attribute value and tracks the change for commit.
func (s *ServerObject) Set(key string, value any) error {
	if _, exists := s.attributes[key]; !exists {
		return fmt.Errorf("attribute %q does not exist", key)
	}

	// Save the original value on first modification only
	if _, tracked := s.oldValues[key]; !tracked {
		old := s.attributes[key]
		// Deep copy slices to prevent aliasing
		if oldSlice, ok := old.([]any); ok {
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

// CommitState returns the current state: "created", "deleted", "changed", or "consistent".
func (s *ServerObject) CommitState() string {
	if s.attributes["object_id"] == nil {
		return "created"
	}
	if s.deleted {
		return "deleted"
	}
	for key, oldVal := range s.oldValues {
		newVal := s.attributes[key]
		if !jsonEqual(oldVal, newVal) {
			return "changed"
		}
	}
	return "consistent"
}

// Rollback reverts all local changes, restoring original attribute values.
func (s *ServerObject) Rollback() {
	s.deleted = false
	for key, oldVal := range s.oldValues {
		s.attributes[key] = oldVal
	}
	s.oldValues = map[string]any{}
}

// serializeChanges builds the change delta for commit payload.
func (s *ServerObject) serializeChanges() map[string]any {
	changes := map[string]any{"object_id": s.Get("object_id")}

	for key, oldVal := range s.oldValues {
		newVal := s.attributes[key]
		if jsonEqual(oldVal, newVal) {
			continue
		}

		oldSlice, oldIsSlice := oldVal.([]any)
		_, newIsSlice := newVal.([]any)

		if oldIsSlice && newIsSlice {
			// Multi-attribute: compute add/remove sets
			newSlice := toAnySlice(newVal)
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
	s.oldValues = map[string]any{}
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

// toAnySlice converts a value known to be a slice to []any.
func toAnySlice(v any) []any {
	if s, ok := v.([]any); ok {
		return s
	}
	return nil
}

// sliceDiff computes elements added to and removed from old to produce new (set semantics).
func sliceDiff(old, cur []any) (add, remove []any) {
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
