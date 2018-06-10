package ping

import (
	"net"
	"sync"
	"time"
)

type request interface {
	init()
	close()
	handleResponse(error, net.IP, *time.Time)
}

// A request is a currently running ICMP echo request waiting for multple answers.
type multiRequest struct {
	tStart    time.Time // when was the request packet sent?
	responses chan Response
	closed    bool
	mtx       sync.RWMutex
}

// Response is a response to a multicast echo request
type Response struct {
	Address  net.IP
	Duration time.Duration
}

// A request is a currently running ICMP echo request waiting for a single answer.
type simpleRequest struct {
	wait    chan struct{}
	result  error
	tStart  time.Time  // when was this packet sent?
	tFinish *time.Time // if and when was the response received?
}

// handleResponse is responsible for finishing this request. It takes an error
// as failure reason.
func (req *simpleRequest) handleResponse(err error, _ net.IP, tRecv *time.Time) {
	req.result = err

	// update tFinish only if no error present and value wasn't previously set
	if err == nil && tRecv != nil && req.tFinish == nil {
		req.tFinish = tRecv
	}
	close(req.wait)
}

func (req *simpleRequest) init() {
	req.wait = make(chan struct{})
	req.tStart = time.Now()
}

func (req *simpleRequest) close() {
	close(req.wait)
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
	req.tStart = time.Now()
}

func (req *multiRequest) close() {
	req.mtx.Lock()
	req.closed = true
	close(req.responses)
	req.mtx.Unlock()
}

// handleResponse is responsible for adding a result to the result set
func (req *multiRequest) handleResponse(_ error, addr net.IP, tRecv *time.Time) {

	// avoid blocking
	go func() {
		req.mtx.RLock()
		defer req.mtx.RUnlock()

		if !req.closed {
			req.responses <- Response{
				Address:  addr,
				Duration: tRecv.Sub(req.tStart),
			}
		}
	}()
}
