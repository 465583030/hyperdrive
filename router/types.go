package router

import (
	"github.com/hyperdrive/router/routerpb"
)

// Route holds the a single route table entry.
type Route struct {
	Destination   string
	InputFilters  []string
	OutputFilters []string
}

// RouteTable is the interface to various route table implementations.
type RouteTable interface {
	Add(source string, route *Route)
	Delete(source string)
	Resolve(source string) (*Route, bool)
}

const (
	msgAddRoute = iota
	msgRemoveRoute
	msgSnapshot
)

type msgType uint

type message struct {
	Type               msgType
	ReplyTo            chan<- interface{}
	AddRouteRequest    routerpb.AddRouteRequest
	RemoveRouteRequest routerpb.RemoveRouteRequest
}
