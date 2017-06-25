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
	readRouterC         <-chan *message
	writeRouterC        chan<- *message
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

func (r *Router) committerEventLoop(ctx context.Context,
	commitC <-chan []byte,
	errorC <-chan error) {

	defer r.eventLoopsWaitGroup.Done()

	for {
		select {
		case data, ok := <-commitC:
			if ok && data != nil {
				var pMsg routerpb.Message
				e := pMsg.Unmarshal(data)
				if e != nil {
					panic(e)
				}

				msg := &message{
					Type:    msgCommit,
					Message: &pMsg,
				}

				r.writeRouterC <- msg
			}
		case e := <-errorC:
			r.log.Error("hyperdrive: committerEventLoop stopped.", e)
			return
		case <-ctx.Done():
			r.log.Error("hyperdrive: committerEventLoop stopped.", ctx.Err())
			return
		}
	}
}

func (r *Router) handleMessage(pPending *map[string]*message,
	msg *message, proposeC chan<- []byte) {
	var msgID uint64
	pending := *pPending

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

		r.log.Debugf("hyperdrive: Proposal %s sent", pmsg.ProposalID)
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
		case routerpb.MsgRemoveRoute:
			r.routeTable.Delete(*pmsg.RemoveRoute.Path)
		}

		// delete the item from map if it was proposed by this node.
		// also notify the waiters.
		if input, ok := pending[pmsg.ProposalID]; ok {
			r.log.Debugf("hyperdrive: Proposal %s is committed", pmsg.ProposalID)
			delete(pending, pmsg.ProposalID)
			input.ReplyTo <- true
			close(input.ReplyTo)
		}

	case msgSnapshot:
		msg.ReplyTo <- []byte{}
	}
}

func (r *Router) routerEventLoop(ctx context.Context,
	proposeC chan<- []byte) {
	defer r.eventLoopsWaitGroup.Done()
	done := false
	pending := map[string]*message{}

	for !done {
		select {
		case e := <-r.readRouterC:
			r.handleMessage(&pending, e, proposeC)
		case <-ctx.Done():
			r.log.Error("hyperdrive: routerEventLoop stopped. ", ctx.Err())
			done = true
		}
	}
}

func (r *Router) snapshotterEventLoop(ctx context.Context,
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
			r.writeRouterC <- m
			s <- (<-resC).([]byte)
		case <-ctx.Done():
			r.log.Error("hyperdrive: snapshotterEventLoop is stopped. ", ctx.Err())
			done = true
		}
	}
}

func (r *Router) closeRouterC() {
	defer r.routerWaitGroup.Done()

	r.eventLoopsWaitGroup.Wait()
	close(r.writeRouterC)
	r.log.Error("hyperdrive: Router internal event loop channel closed.")
}

// CreateSnapshot generates a snapshot for recovery.
func (r *Router) CreateSnapshot() ([]byte, error) {
	return []byte{}, nil
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
	commitC <-chan []byte,
	errorC <-chan error,
	routeTable RouteTable,
	logger *logrus.Logger) *Router {

	eventLoopsWaitGroup := &sync.WaitGroup{}
	routerWaitGroup := &sync.WaitGroup{}

	routerC := make(chan *message)

	n := &Router{
		port:                port,
		apiPort:             apiPort,
		snapshotC:           snapshotC,
		log:                 logger,
		eventLoopsWaitGroup: eventLoopsWaitGroup,
		routerWaitGroup:     routerWaitGroup,
		readRouterC:         routerC,
		writeRouterC:        routerC,
		routeTable:          routeTable,
	}

	routerWaitGroup.Add(1)
	eventLoopsWaitGroup.Add(3)

	go n.routerEventLoop(ctx, proposeC)
	go n.snapshotterEventLoop(ctx, snapshotC)
	go n.committerEventLoop(ctx, commitC, errorC)
	go n.closeRouterC()

	// TODO: Wait until the node is correctly registered with raft.
	// go r.startAPIService()
	// r.startRouterService()
	return n
}

func (r *Router) sendCore(msg *message) <-chan interface{} {
	replyTo := make(chan interface{}, 1)
	msg.ReplyTo = replyTo
	r.writeRouterC <- msg
	return replyTo
}

/*
AddNewRoute adds the suggested route to the table.
This function returns before the route is committed.

Use the returned channel to determine if the route was
successfully added or not.
*/
func (r *Router) AddNewRoute(
	req *routerpb.AddRouteRequest) <-chan interface{} {
	return r.sendCore(&message{
		Type:            msgProposeAddRoute,
		AddRouteRequest: req,
	})
}

/*
RemoveRoute removes the suggested route from the table.
This function returns before the route is committed.

Use the returned channel to determine if the route was
successfully removed or not.
*/
func (r *Router) RemoveRoute(
	req *routerpb.RemoveRouteRequest) <-chan interface{} {
	return r.sendCore(&message{
		Type:               msgProposeRemoveRoute,
		RemoveRouteRequest: req,
	})
}
