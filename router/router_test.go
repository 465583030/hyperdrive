package router

import (
	"context"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/sirupsen/logrus/hooks/null"
	"github.com/stretchr/testify/assert"
)

func TestResponseToSnapshotRequests(t *testing.T) {
	logger, _ := null.NewNullLogger()
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*2)
	defer cancel()

	snapshotC := make(chan chan<- []byte)
	NewRouter(ctx, 80, 81, nil, snapshotC, logger)

	resC := make(chan []byte)
	snapshotC <- resC
	select {
	case snap := <-resC:
		if len(snap) != 0 {
			t.Fatal("expected a zero length snapshot but received more")
		}
		t.Log("response received")
	case <-ctx.Done():
		t.Fatal("timeout expired. did not receive a response")
	}
}

func TestLoggingWhenExitingEventLoop(t *testing.T) {
	logger, hook := null.NewNullLogger()
	ctx, cancel := context.WithCancel(context.Background())

	snapshotC := make(chan chan<- []byte)
	n := NewRouter(ctx, 80, 81, nil, snapshotC, logger)
	cancel()
	n.WaitForExit()

	assert.Equal(t, 2, len(hook.Entries))
	for _, e := range hook.Entries {
		assert.Equal(t, logrus.ErrorLevel, e.Level)
	}
}
