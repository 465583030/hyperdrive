package router

// Route contains the configuration info for a route.
type Route struct {
	Destination   string
	InputFilters  []string
	OutputFilters []string
}

// RouteTable is the common interface implemented by all route tables.
type RouteTable interface {
	Add(source string, route *Route)
	Delete(source string)
	Resolve(source string) (*Route, bool)
}
