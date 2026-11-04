package ping

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestSimpleRequestFinishesOnce(t *testing.T) {
	assert := assert.New(t)

	req := simpleRequest{}
	req.init()

	tRecv := req.tStart.Add(time.Millisecond)
	req.handleReply(nil, nil, &tRecv)
	req.handleReply(errClosed, nil, nil) // must neither panic nor override the result
	req.close()

	rtt, err := req.roundTripTime()
	assert.NoError(err)
	assert.Equal(time.Millisecond, rtt)
}

func TestMultiRequestCloseWithUnreadReplies(t *testing.T) {
	req := multiRequest{}
	req.init()

	now := time.Now()
	req.handleReply(nil, nil, &now)
	req.handleReply(nil, nil, &now)

	done := make(chan struct{})
	go func() {
		req.close()
		req.close()
		req.handleReply(nil, nil, &now) // ignored after close
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("close blocked on unread replies")
	}

	for range req.replies { // drain
	}
}

func TestMultiRequestDeliversReplies(t *testing.T) {
	req := multiRequest{}
	req.init()

	tRecv := req.tStart.Add(time.Millisecond)
	req.handleReply(nil, nil, &tRecv)
	req.handleReply(errors.New("ignored"), nil, nil)

	reply := <-req.replies
	assert.Equal(t, time.Millisecond, reply.Duration)

	req.close()
	_, ok := <-req.replies
	assert.False(t, ok)
}
