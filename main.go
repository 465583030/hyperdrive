package main

import (
	"flag"
	"strings"

	"github.com/coreos/etcd/raft/raftpb"
	"github.com/hyperdrive/raft"
	"github.com/hyperdrive/router"
)

func main() {
	cluster := flag.String("cluster", "http://127.0.0.1:8211", "comma separated cluster peers")

	id := flag.Int("id", 1, "node ID")
	port := flag.Int("port", 8311, "router port")
	apiPort := flag.Int("api-port", 8312, "api endpoint port")
	join := flag.Bool("join", false, "join an existing cluster")
	flag.Parse()

	proposeC := make(chan []byte)
	defer close(proposeC)
	confChangeC := make(chan raftpb.ConfChange)
	defer close(confChangeC)

	router := router.CreateNewRouter()

	// raft provides a commit stream for the proposals from the http api
	commitC, errorC, snapshotterReady := raft.NewNode(*id, strings.Split(*cluster, ","), *join, router.CreateSnapshot, proposeC, confChangeC)

	router.Start(*port, *apiPort, commitC, errorC, snapshotterReady)
}
