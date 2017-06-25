package router

import (
	"context"
	"testing"
	"time"

	"github.com/hyperdrive/router/routerpb"
	"github.com/sirupsen/logrus"
	"github.com/sirupsen/logrus/hooks/null"
	"github.com/stretchr/testify/assert"
)

type testRouterContext struct {
	logger     *logrus.Logger
	hook       *null.Hook
	context    context.Context
	proposeC   chan []byte
	snapshotC  chan chan<- []byte
	commitC    chan []byte
	errorC     chan error
	routeTable RouteTable
}

func newTestRouter(ctx context.Context) (*Router, *testRouterContext) {
	if ctx == nil {
		ctx = context.TODO()
	}

	logger, hook := null.NewNullLogger()

	tctx := &testRouterContext{
		logger:     logger,
		hook:       hook,
		context:    ctx,
		proposeC:   make(chan []byte),
		snapshotC:  make(chan chan<- []byte),
		commitC:    make(chan []byte),
		errorC:     make(chan error),
		routeTable: NewMapRouteTable(),
	}

	router := NewRouter(tctx.context,
		80,
		81,
		nil,
		tctx.proposeC,
		tctx.snapshotC,
		tctx.commitC,
		tctx.errorC,
		tctx.routeTable,
		tctx.logger)

	return router, tctx
}

func TestResponseToSnapshotRequests(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*2)
	defer cancel()

	_, tc := newTestRouter(ctx)

	resC := make(chan []byte)
	tc.snapshotC <- resC

	select {
	case snap := <-resC:
		if len(snap) != 0 {
			t.Fatal("expected a zero length snapshot but received more")
		}
		t.Log("response received")
	case <-tc.context.Done():
		t.Fatal("timeout expired. did not receive a response")
	}
}

func TestLoggingWhenExitingEventLoop(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())

	n, tc := newTestRouter(ctx)
	cancel()
	n.WaitForExit()

	assert.Equal(t, 4, len(tc.hook.Entries))
	for _, e := range tc.hook.Entries {
		assert.Equal(t, logrus.ErrorLevel, e.Level)
	}
}

func TestAddNewRoute(t *testing.T) {
	path := "a"
	route := "b"

	r := &routerpb.AddRouteRequest{
		Path:  &path,
		Route: &route,
	}

	n, tc := newTestRouter(nil)
	d := n.AddNewRoute(r)

	p := <-tc.proposeC
	tc.commitC <- p

	<-d

	sr, ok := tc.routeTable.Resolve("a")
	assert.True(t, ok)
	assert.Equal(t, "b", sr.Destination)
}

func TestRemoveRoute(t *testing.T) {
	path := "a"
	route := "b"

	n, tc := newTestRouter(nil)
	d := n.AddNewRoute(&routerpb.AddRouteRequest{
		Path:  &path,
		Route: &route,
	})

	tc.commitC <- <-tc.proposeC
	<-d

	d = n.RemoveRoute(&routerpb.RemoveRouteRequest{
		Path: &path,
	})

	tc.commitC <- <-tc.proposeC
	<-d

	_, ok := tc.routeTable.Resolve("a")
	assert.False(t, ok)
}
