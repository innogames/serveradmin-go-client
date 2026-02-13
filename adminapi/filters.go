package adminapi

type (
	Filters map[string]any
	Filter  map[string]any
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

func Regexp(value string) Filter {
	return createFilter("Regexp", value)
}

func Not[V valueOrFilter](filter V) Filter {
	return createFilter("Not", filter)
}

func NotEmpty() Filter {
	return Not(Empty())
}

func Any[V valueOrFilter](values ...V) Filter {
	return createFilter("Any", values)
}

func All[V valueOrFilter](values ...V) Filter {
	return createFilter("All", values)
}

func Empty() Filter {
	return createFilter("Empty", nil)
}

func StartsWith(value string) Filter {
	return createFilter("StartsWith", value)
}

func GreaterThan[N int | float64](value N) Filter {
	return createFilter("GreaterThan", value)
}

func GreaterThanOrEquals[N int | float64](value N) Filter {
	return createFilter("GreaterThanOrEquals", value)
}

func LessThan[N int | float64](value N) Filter {
	return createFilter("LessThan", value)
}

func LessThanOrEquals[N int | float64](value N) Filter {
	return createFilter("LessThanOrEquals", value)
}

func Contains[V valueOrFilter](value V) Filter {
	return createFilter("Contains", value)
}

func ContainedBy[V valueOrFilter](value V) Filter {
	return createFilter("ContainedBy", value)
}

func ContainedOnlyBy[V valueOrFilter](value V) Filter {
	return createFilter("ContainedOnlyBy", value)
}

func Overlaps[V valueOrFilter](value V) Filter {
	return createFilter("Overlaps", value)
}

func createFilter(filterType string, value any) Filter {
	return Filter{
		filterType: value,
	}
}
