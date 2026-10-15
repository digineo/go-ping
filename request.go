package ping

import (
	"net"
	"sync"
	"time"
)

type request interface {
	init()
	close()
	handleReply(error, net.IP, *time.Time)
}

// A multiRequest is a currently running ICMP echo request waiting for multple answers.
type multiRequest struct {
	tStart  time.Time // when was the request packet sent?
	replies chan Reply
	done    chan struct{}  // closed when the request is finished
	pending sync.WaitGroup // replies currently being delivered
	closed  bool
	mtx     sync.Mutex
}

// Reply is a reply to a multicast echo request
type Reply struct {
	Address  net.IP
	Duration time.Duration
}

// A simpleRequest is a currently running ICMP echo request waiting for a single answer.
type simpleRequest struct {
	wait    chan struct{}
	once    sync.Once
	result  error
	tStart  time.Time  // when was this packet sent?
	tFinish *time.Time // if and when was the reply received?
}

// handleReply is responsible for finishing this request.
// It takes an error as failure reason. Only the first call has an effect.
func (req *simpleRequest) handleReply(err error, _ net.IP, tRecv *time.Time) {
	req.once.Do(func() {
		req.result = err
		if err == nil {
			req.tFinish = tRecv
		}
		close(req.wait)
	})
}

func (req *simpleRequest) init() {
	req.wait = make(chan struct{})
	req.tStart = time.Now()
}

func (req *simpleRequest) close() {
	// A race condition may happen when sending fails and a reply is
	// received anyway, handleReply ensures we only close once.
	req.handleReply(nil, nil, nil)
}

func (req *simpleRequest) roundTripTime() (time.Duration, error) {
	if req.result != nil {
		return 0, req.result
	}
	if req.tFinish == nil {
		return 0, nil
	}
	return req.tFinish.Sub(req.tStart), nil
}

func (req *multiRequest) init() {
	req.replies = make(chan Reply)
	req.done = make(chan struct{})
	req.tStart = time.Now()
}

// close finishes the request. Pending replies are discarded and the
// replies channel is closed. It is safe to call close multiple times.
func (req *multiRequest) close() {
	req.mtx.Lock()
	if req.closed {
		req.mtx.Unlock()
		return
	}
	req.closed = true
	close(req.done)
	req.mtx.Unlock()

	req.pending.Wait()
	close(req.replies)
}

// handleReply is responsible for adding a result to the result set
func (req *multiRequest) handleReply(err error, addr net.IP, tRecv *time.Time) {
	if err != nil {
		return
	}

	req.mtx.Lock()
	defer req.mtx.Unlock()
	if req.closed {
		return
	}

	reply := Reply{
		Address:  addr,
		Duration: tRecv.Sub(req.tStart),
	}

	// avoid blocking the receiver
	req.pending.Add(1)
	go func() {
		defer req.pending.Done()
		select {
		case req.replies <- reply:
		case <-req.done:
		}
	}()
}
