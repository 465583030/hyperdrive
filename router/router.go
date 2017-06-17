package router

import (
	"context"
	"fmt"
	"net/http"

	"log"

	"sync"

	"github.com/hyperdrive/raft"
	"github.com/hyperdrive/router/routerpb"
	"github.com/sirupsen/logrus"
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
	msgProposeAddRoute = iota
	msgProposeRemoveRoute
	msgCommit
	msgSnapshot
)

type msgType uint

type message struct {
	Type               msgType
	ReplyTo            chan<- interface{}
	AddRouteRequest    *routerpb.AddRouteRequest
	RemoveRouteRequest *routerpb.RemoveRouteRequest
	Message            *routerpb.Message
}

// Router is the state of router component.
type Router struct {
	id                  string
	port                int
	apiPort             int
	node                *raft.Node
	snapshotC           <-chan chan<- []byte
	routeTable          RouteTable
	log                 *logrus.Logger
	eventLoopsWaitGroup *sync.WaitGroup
	routerWaitGroup     *sync.WaitGroup
}

// CreateSnapshot generates a snapshot for recovery.
func (r *Router) CreateSnapshot() ([]byte, error) {
	return []byte{}, nil
}

func invokeFilter(req *http.Request, filter string) {
}

func (r *Router) handleRoute(res http.ResponseWriter, req *http.Request) {
	route, ok := r.routeTable.Resolve(req.URL.Path)
	if !ok {
		res.WriteHeader(http.StatusNotFound)
		return
	}

	for _, in := range route.InputFilters {
		invokeFilter(req, in)
	}
}

func (r *Router) handleAPIRoute(res http.ResponseWriter, req *http.Request) {
	res.WriteHeader(http.StatusOK)
}

func (r *Router) startRouterService() {
	routerService := http.NewServeMux()
	routerService.HandleFunc("/", r.handleRoute)
	err := http.ListenAndServe(fmt.Sprintf(":%d", r.port), routerService)
	if err != nil {
		panic(err)
	}
}

func (r *Router) startAPIService() {
	apiService := http.NewServeMux()
	apiService.HandleFunc("/routes", r.handleAPIRoute)
	log.Println("a")
	err := http.ListenAndServe(fmt.Sprintf(":%d", r.apiPort), apiService)
	if err != nil {
		panic(err)
	}
}

func (r *Router) readCommits() {
	for {
		select {
		case data := <-r.node.Commits():
			if data != nil {
				log.Printf(string(data))
			}
		case <-r.node.Errors():
			break
		}
	}
}

func (r *Router) handleMessage(msg *message, proposeC chan<- []byte) {
	pending := map[string]*message{}
	var msgID uint64

	switch msg.Type {
	case msgProposeAddRoute, msgProposeRemoveRoute:
		msgID++
		pmsg := &routerpb.Message{
			ProposalID: fmt.Sprintf("%s-%d", r.id, msgID),
		}

		switch msg.Type {
		case msgProposeAddRoute:
			pmsg.Type = routerpb.MsgAddRoute
			pmsg.AddRoute = msg.AddRouteRequest
		case msgProposeRemoveRoute:
			pmsg.Type = routerpb.MsgRemoveRoute
			pmsg.RemoveRoute = msg.RemoveRouteRequest
		}

		pending[pmsg.ProposalID] = msg

		go func() {
			payload, err := pmsg.Marshal()
			// (buddyspike) what do we do with err?
			if err == nil {
				proposeC <- payload
			}
		}()
	case msgCommit:
		pmsg := msg.Message
		switch pmsg.Type {
		case routerpb.MsgAddRoute:
			r.routeTable.Add(*pmsg.AddRoute.Path, &Route{
				Destination:   *pmsg.AddRoute.Route,
				InputFilters:  pmsg.AddRoute.Preprocessors,
				OutputFilters: pmsg.AddRoute.Postprocessors,
			})
		}
		// delete the item from map if it was proposed by this node.
		// also notify the waiters.
		if input, ok := pending[pmsg.ProposalID]; ok {
			delete(pending, pmsg.ProposalID)
			input.ReplyTo <- true
		}

	case msgSnapshot:
		msg.ReplyTo <- []byte{}
	}
}

func (r *Router) routerEventLoop(ctx context.Context,
	input <-chan *message,
	proposeC chan<- []byte) {
	defer r.eventLoopsWaitGroup.Done()
	done := false

	for !done {
		select {
		case e := <-input:
			r.handleMessage(e, proposeC)
		case <-ctx.Done():
			r.log.Error("hyperdrive: Router event loop stopped. ", ctx.Err())
			done = true
		}
	}
}

func (r *Router) snapshotterEventLoop(ctx context.Context,
	routerC chan<- *message,
	snapshotC <-chan chan<- []byte) {

	defer r.eventLoopsWaitGroup.Done()
	done := false

	for !done {
		select {
		case s := <-snapshotC:
			resC := make(chan interface{})
			m := &message{
				Type:    msgSnapshot,
				ReplyTo: resC,
			}
			routerC <- m
			s <- (<-resC).([]byte)
		case <-ctx.Done():
			r.log.Error("hyperdrive: Snapshotter event loop is stopped. ", ctx.Err())
			done = true
		}
	}
}

func (r *Router) closeRouterC(routerC chan *message) {
	defer r.routerWaitGroup.Done()

	r.eventLoopsWaitGroup.Wait()
	close(routerC)
	r.log.Error("hyperdrive: Router internal event loop channel closed.")
}

// WaitForExit waits until all goroutines of the router returns.
func (r *Router) WaitForExit() {
	r.routerWaitGroup.Wait()
}

// NewRouter starts the routing module.
// Close ctx to notify the clean-up.
// Closing of proposeC and snapshotC should be done after
// WaitForExit returns.
func NewRouter(ctx context.Context,
	port int,
	apiPort int,
	raftNode *raft.Node,
	proposeC chan<- []byte,
	snapshotC <-chan chan<- []byte,
	logger *logrus.Logger) *Router {

	eventLoopsWaitGroup := &sync.WaitGroup{}
	routerWaitGroup := &sync.WaitGroup{}

	n := &Router{
		port:                port,
		apiPort:             apiPort,
		snapshotC:           snapshotC,
		log:                 logger,
		eventLoopsWaitGroup: eventLoopsWaitGroup,
		routerWaitGroup:     routerWaitGroup,
	}

	routerWaitGroup.Add(1)
	eventLoopsWaitGroup.Add(2)

	routerC := make(chan *message)

	go n.routerEventLoop(ctx, routerC, proposeC)
	go n.snapshotterEventLoop(ctx, routerC, snapshotC)
	go n.closeRouterC(routerC)
	// go r.readCommits()

	// TODO: Wait until the node is correctly registered with raft.
	// go r.startAPIService()
	// r.startRouterService()
	return n
}
