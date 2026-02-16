package adminapi

// MultiAttr is a helper type for multi-valued attributes.
// It provides set-like operations on string slices.
//
// MultiAttr maintains set semantics: Add prevents duplicates, Delete removes all
// occurrences. Changes are made in-place but do NOT automatically update a ServerObject.
// Users must call obj.Set() manually after modifications.
//
// Example usage:
//
//	tags := serverObject.GetMulti("tags")
//	tags.Add("web", "prod")
//	tags.Delete("old-tag")
//	obj.Set("tags", tags)
type MultiAttr []string

// Add appends elements to the MultiAttr, preventing duplicates (set semantics).
// If an element already exists, it is not added again. Handles nil receiver
// by allocating a new slice.
//
// Example:
//
//	m := MultiAttr{"web", "prod"}
//	m.Add("api", "web")  // Only "api" is added (web already exists)
//	// m is now ["web", "prod", "api"]
func (m *MultiAttr) Add(elems ...string) {
	for _, elem := range elems {
		if !m.Contains(elem) {
			*m = append(*m, elem)
		}
	}
}

// Delete removes all occurrences of the element from the MultiAttr.
// If the element doesn't exist, this is a no-op.
//
// Example:
//
//	m := MultiAttr{"web", "prod", "web"}
//	m.Delete("web")
//	// m is now ["prod"]
func (m *MultiAttr) Delete(elem string) {
	filtered := make(MultiAttr, 0, len(*m))
	for _, v := range *m {
		if v != elem {
			filtered = append(filtered, v)
		}
	}
	*m = filtered
}

// Clear removes all elements from the MultiAttr, resulting in an empty slice.
//
// Example:
//
//	m := MultiAttr{"web", "prod"}
//	m.Clear()
//	// m is now []
func (m *MultiAttr) Clear() {
	*m = MultiAttr{}
}

// Contains returns true if the element exists in the MultiAttr.
// Returns false for nil or empty slices.
//
// Example:
//
//	m := MultiAttr{"web", "prod"}
//	m.Contains("web")   // true
//	m.Contains("api")   // false
func (m MultiAttr) Contains(elem string) bool {
	for _, v := range m {
		if v == elem {
			return true
		}
	}
	return false
}
