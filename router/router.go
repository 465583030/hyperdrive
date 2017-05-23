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
}

// CreateSnapshot generates a snapshot for recovery.
func (r *Router) CreateSnapshot() ([]byte, error) {
	return []byte{}, nil
}

// HandleRoute dispatches the incoming request to the appropriate route.
func (r *Router) HandleRoute(res http.ResponseWriter, req *http.Request) {
	res.Write([]byte("hello\n"))
}

// HandleAPIRoute is the entrypoint for route management api
func (r *Router) HandleAPIRoute(res http.ResponseWriter, req *http.Request) {
	res.WriteHeader(http.StatusOK)
}

func (r *Router) startRouterService(port int) {
	routerService := http.NewServeMux()
	routerService.HandleFunc("/", r.HandleRoute)
	http.ListenAndServe(fmt.Sprintf(":%d", port), routerService)
}

func (r *Router) startAPIService(port int) {
	apiService := http.NewServeMux()
	apiService.HandleFunc("/routes", r.HandleAPIRoute)
	http.ListenAndServe(fmt.Sprintf(":%d", port), apiService)
}

func (r *Router) readCommits(commitC <-chan *string, errorC <-chan error) {
	for {
		select {
		case data := <-commitC:
			if data != nil {
				log.Printf(*data)
			}
		case <-errorC:
			break
		}
	}
}

// Start serving routing requests.
func (r *Router) Start(port int, apiPort int, commitC <-chan *string, errorC <-chan error, snapshotterReady <-chan *snap.Snapshotter) {
	go r.readCommits(commitC, errorC)

	// TODO: Wait until the node is correctly registered with raft.
	go r.startAPIService(apiPort)
	r.startRouterService(port)
}

// CreateNewRouter creates a new router.
func CreateNewRouter() *Router {
	return &Router{}
}
