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

func (r *Router) startRouterService(port int) {
	routerService := http.NewServeMux()
	routerService.HandleFunc("/", r.handleRoute)
	err := http.ListenAndServe(fmt.Sprintf(":%d", port), routerService)
	if err != nil {
		panic(err)
	}
}

func (r *Router) startAPIService(port int) {
	apiService := http.NewServeMux()
	apiService.HandleFunc("/routes", r.handleAPIRoute)
	log.Println("a")
	err := http.ListenAndServe(fmt.Sprintf(":%d", port), apiService)
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
func (r *Router) Start(port int, apiPort int) {
	go r.readCommits()

	// TODO: Wait until the node is correctly registered with raft.
	go r.startAPIService(apiPort)
	r.startRouterService(port)
}

// CreateNewRouter creates a new router.
func CreateNewRouter(commitC <-chan *string, errorC <-chan error, snapshotterReady <-chan *snap.Snapshotter) *Router {
	return &Router{
		commitC:          commitC,
		errorC:           errorC,
		snapshotterReady: snapshotterReady,
	}
}
