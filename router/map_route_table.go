package router

type MapRouteTable struct {
	table map[string]*Route
}

func NewMapRouteTable() *MapRouteTable {
	return &MapRouteTable{
		table: map[string]*Route{},
	}
}

func (t *MapRouteTable) Add(source string, route *Route) {
	t.table[source] = route
}

func (t *MapRouteTable) Delete(source string) {
	delete(t.table, source)
}

func (t *MapRouteTable) Resolve(source string) (*Route, bool) {
	r, ok := t.table[source]
	return r, ok
}
