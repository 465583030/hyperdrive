package router

import (
	"context"
	"fmt"
	"net/http"

	"log"

	"sync"

	"github.com/hyperdrive/raft"
	"github.com/sirupsen/logrus"
)

// Node is the external view of a router node.
type Node struct {
	log       *logrus.Logger
	waitGroup *sync.WaitGroup
}

// Router is the state of router component.
type Router struct {
	port       int
	apiPort    int
	node       *raft.Node
	snapshotC  <-chan chan<- []byte
	routeTable RouteTable
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

func handleMessage(msg *message) {
	switch msg.Type {
	case msgAddRoute:
		// TODO add route implementation
	case msgRemoveRoute:
		// TODO remove route implemetation
	case msgSnapshot:
		msg.ReplyTo <- []byte{}
	}
}

func (n *Node) routerEventLoop(ctx context.Context, input <-chan *message) {
	defer n.waitGroup.Done()
	done := false

	for !done {
		select {
		case e := <-input:
			handleMessage(e)
		case <-ctx.Done():
			n.log.Error("hyperdrive: Router event loop stopped. ", ctx.Err())
			done = true
		}
	}
}

func (n *Node) snapshotterEventLoop(ctx context.Context,
	routerC chan<- *message,
	snapshotC <-chan chan<- []byte) {

	defer n.waitGroup.Done()
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
			n.log.Error("hyperdrive: Snapshotter event loop is stopped. ", ctx.Err())
			done = true
		}
	}
}

// WaitForExit waits until all goroutines of the router returns.
func (n *Node) WaitForExit() {
	n.waitGroup.Wait()
}

// NewRouter starts the routing module.
func NewRouter(ctx context.Context,
	port int,
	apiPort int,
	node *raft.Node,
	snapshotC <-chan chan<- []byte,
	logger *logrus.Logger) *Node {

	wg := &sync.WaitGroup{}

	n := &Node{
		log:       logger,
		waitGroup: wg,
	}

	wg.Add(2)

	routerC := make(chan *message)

	go n.routerEventLoop(ctx, routerC)
	go n.snapshotterEventLoop(ctx, routerC, snapshotC)
	// go r.readCommits()

	// TODO: Wait until the node is correctly registered with raft.
	// go r.startAPIService()
	// r.startRouterService()
	return n
}
