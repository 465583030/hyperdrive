package router

import (
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

// Start serving routing requests.
func (r *Router) Start(port int, commitC <-chan *string, errorC <-chan error, snapshotterReady <-chan *snap.Snapshotter) {
	go readCommits(commitC, errorC)

	http.HandleFunc("/", func(res http.ResponseWriter, req *http.Request) {
		res.Write([]byte("hello\n"))
	})

	http.ListenAndServe(":8080", nil)
}

func readCommits(commitC <-chan *string, errorC <-chan error) {
	for data := range commitC {
		if data != nil {
			log.Printf(*data)
		}
	}
}

// CreateNewRouter creates a new router.
func CreateNewRouter() *Router {
	return &Router{}
}
