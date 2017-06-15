package router

import (
	"context"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/sirupsen/logrus/hooks/null"
	"github.com/stretchr/testify/assert"
)

type testRouterContext struct {
	logger    *logrus.Logger
	hook      *null.Hook
	context   context.Context
	proposeC  chan []byte
	snapshotC chan chan<- []byte
}

func newTestRouter(ctx context.Context) (*Router, *testRouterContext) {
	if ctx == nil {
		ctx = context.TODO()
	}

	logger, hook := null.NewNullLogger()

	testContext := &testRouterContext{
		logger:    logger,
		hook:      hook,
		context:   ctx,
		proposeC:  make(chan []byte),
		snapshotC: make(chan chan<- []byte),
	}

	router := NewRouter(testContext.context,
		80, 81, nil, testContext.proposeC, testContext.snapshotC, testContext.logger)

	return router, testContext
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

	assert.Equal(t, 2, len(tc.hook.Entries))
	for _, e := range tc.hook.Entries {
		assert.Equal(t, logrus.ErrorLevel, e.Level)
	}
}
