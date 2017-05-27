package router

import (
	"fmt"
	"net/http"

	"log"

	"github.com/coreos/etcd/snap"
)

// Router is the state of router component.
type Router struct {
	port             int
	apiPort          int
	commitC          <-chan *string
	errorC           <-chan error
	snapshotterReady <-chan *snap.Snapshotter
	routeTable       RouteTable
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
		case data := <-r.commitC:
			if data != nil {
				log.Printf(*data)
			}
		case <-r.errorC:
			break
		}
	}
}

// Start serving routing requests.
func (r *Router) Start(port int, apiPort int, commitC <-chan *string, errorC <-chan error, snapshotterReady <-chan *snap.Snapshotter) {
	r.port = port
	r.apiPort = apiPort
	r.commitC = commitC
	r.errorC = errorC
	r.snapshotterReady = snapshotterReady

	go r.readCommits()

	// TODO: Wait until the node is correctly registered with raft.
	go r.startAPIService()
	r.startRouterService()
}

// CreateNewRouter creates a new router.
func CreateNewRouter() *Router {
	return &Router{}
}
