package adminapi

type (
	// Filters maps attribute names to filter values or Filter objects.
	// Used as the top-level query predicate: Filters{"hostname": Regexp("web.*"), "state": "online"}.
	Filters map[string]any

	// Filter represents a single filter operation like Regexp, Not, or Any.
	// Unlike Filters, a Filter wraps a named operation with its argument.
	Filter map[string]any
)

type value interface {
	int | string | bool
}
type valueOrFilter interface {
	value | Filter
}

// list of all valid functions with lowercased key
var allFilters = map[string]string{
	"any":                 "Any",
	"all":                 "All",
	"containedby":         "ContainedBy",
	"containedonlyby":     "ContainedOnlyBy",
	"contains":            "Contains",
	"empty":               "Empty",
	"greaterthan":         "GreaterThan",
	"greaterthanorequals": "GreaterThanOrEquals",
	"lessthan":            "LessThan",
	"lessthanorequals":    "LessThanOrEquals",
	"not":                 "Not",
	"overlaps":            "Overlaps",
	"regexp":              "Regexp",
	"startswith":          "StartsWith",
}

// Not creates a filter that negates the given filter or value. For example, Not(2) means "!= 2".
func Not[V valueOrFilter](filter V) Filter {
	return createFilter("Not", filter)
}

// NotEmpty is a shortcut function for Not(Empty())
func NotEmpty() Filter {
	return Not(Empty())
}

// Empty matches attributes that have no value (nil or empty).
func Empty() Filter {
	return createFilter("Empty", nil)
}

// Any matches if the attribute matches any of the given values or filters (OR semantics).
func Any[V valueOrFilter](values ...V) Filter {
	return createFilter("Any", values)
}

// All matches if the attribute matches all of the given values or filters (AND semantics).
func All[V valueOrFilter](values ...V) Filter {
	return createFilter("All", values)
}

// Regexp matches the attribute value against the given regular expression pattern.
func Regexp(value string) Filter {
	return createFilter("Regexp", value)
}

// StartsWith matches attributes whose value begins with the given prefix.
func StartsWith(value string) Filter {
	return createFilter("StartsWith", value)
}

// GreaterThan matches attributes with a numeric value strictly greater than the given value.
func GreaterThan(value int) Filter {
	return createFilter("GreaterThan", value)
}

// GreaterThanOrEquals matches attributes with a numeric value greater than or equal to the given value.
func GreaterThanOrEquals(value int) Filter {
	return createFilter("GreaterThanOrEquals", value)
}

// LessThan matches attributes with a numeric value strictly less than the given value.
func LessThan(value int) Filter {
	return createFilter("LessThan", value)
}

// LessThanOrEquals matches attributes with a numeric value less than or equal to the given value.
func LessThanOrEquals(value int) Filter {
	return createFilter("LessThanOrEquals", value)
}

// Contains matches multi-valued attributes that contain the given value.
func Contains[V valueOrFilter](value V) Filter {
	return createFilter("Contains", value)
}

// ContainedBy matches attributes whose values are a subset of the given value.
func ContainedBy[V valueOrFilter](value V) Filter {
	return createFilter("ContainedBy", value)
}

// ContainedOnlyBy matches attributes whose values are exclusively contained by the given value.
func ContainedOnlyBy[V valueOrFilter](value V) Filter {
	return createFilter("ContainedOnlyBy", value)
}

// Overlaps matches multi-valued attributes that share at least one element with the given value.
func Overlaps[V valueOrFilter](value V) Filter {
	return createFilter("Overlaps", value)
}

func createFilter(filterType string, value any) Filter {
	return Filter{
		filterType: value,
	}
}
