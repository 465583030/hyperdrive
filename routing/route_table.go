package routing

import (
	"strings"
)

// RouteTable is the core route table implementation.
type RouteTable struct {
	root *Route
}

// Add creates a new entry in RouteTable.
func (*RouteTable) Add(source string, destination string) {

}

// Remove deletes an entry from RouteTable.
func (*RouteTable) Remove(source string) {

}

func resolveRecursive(parts []string, start *RouteTable) *Resolution {
	return nil
}

// Resolve finds the matching route for a given path.
func (rt *RouteTable) Resolve(source string) (*Resolution, error) {
	parts := strings.Split(source, "/")
	return resolveRecursive(parts, rt), nil
}
