package main

import (
	"context"
	"flag"
	"strings"

	"github.com/coreos/etcd/raft/raftpb"
	"github.com/hyperdrive/raft"
	"github.com/hyperdrive/router"
	"github.com/sirupsen/logrus"
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
	snapshotC := make(chan chan<- []byte)
	commitC := make(chan []byte)
	defer close(commitC)
	errorC := make(chan error)
	defer close(errorC)

	rt := router.NewMapRouteTable()
	node := raft.NewNode(*id,
		strings.Split(*cluster, ","),
		*join,
		snapshotC,
		proposeC,
		commitC,
		errorC,
		confChangeC)

	logger := &logrus.Logger{}
	router.NewRouter(context.TODO(),
		*port,
		*apiPort,
		node,
		proposeC,
		snapshotC,
		commitC,
		errorC,
		rt,
		logger)
}
