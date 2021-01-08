package ping

import (
	"context"
	"errors"
	"net"
	"sync/atomic"
	"time"
)

// sequence number for this process
var sequence uint32

// PingAttempts sends ICMP echo requests with a timeout per request, retrying upto `attempt` times .
// Will finish early on success and return the round trip time of the last ping.
func (pinger *Pinger) PingAttempts(destination *net.IPAddr, timeout time.Duration, attempts int) (rtt time.Duration, err error) {
	if attempts < 1 {
		err = errors.New("zero attempts")
	} else {
		for i := 0; i < attempts; i++ {
			rtt, err = pinger.Ping(destination, timeout)
			if err == nil {
				break // success
			}
		}
	}
	return
}

// Ping sends a single Echo Request and waits for an answer. It returns
// the round trip time (RTT) if a reply is received in time.
func (pinger *Pinger) Ping(destination *net.IPAddr, timeout time.Duration) (time.Duration, error) {
	ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(timeout))
	defer cancel()
	return pinger.PingContext(ctx, destination)
}

// PingContext sends a single Echo Request and waits for an answer. It returns
// the round trip time (RTT) if a reply is received before cancellation of the context.
func (pinger *Pinger) PingContext(ctx context.Context, destination *net.IPAddr) (time.Duration, error) {
	req := simpleRequest{}

	seq, err := pinger.sendRequest(destination, &req)
	if err != nil {
		return 0, err
	}

	// wait for answer
	select {
	case <-req.wait:
		// already dequeued
		err = req.result
	case <-ctx.Done():
		// dequeue request
		pinger.removeRequest(seq)
		err = &timeoutError{}
	}

	if err != nil {
		return 0, err
	}
	return req.roundTripTime()
}

// PingMulticast sends a single echo request and returns a channel for the responses.
// The channel will be closed on termination of the context.
// An error is returned if the sending of the echo request fails.
func (pinger *Pinger) PingMulticast(destination *net.IPAddr, wait time.Duration) (<-chan Reply, error) {
	ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(wait))
	defer cancel()
	return pinger.PingMulticastContext(ctx, destination)
}

// PingMulticastContext does the same as PingMulticast but receives a context
func (pinger *Pinger) PingMulticastContext(ctx context.Context, destination *net.IPAddr) (<-chan Reply, error) {
	req := multiRequest{}

	seq, err := pinger.sendRequest(destination, &req)
	if err != nil {
		return nil, err
	}

	go func() {
		<-ctx.Done()

		// dequeue request
		pinger.removeRequest(seq)

		req.close()
	}()

	return req.replies, nil
}

// sendRequest marshals the payload and sends the packet.
// It returns the sequence number and an error if the sending failed.
func (pinger *Pinger) sendRequest(dest *net.IPAddr, req request) (uint16, error) {
	seq := uint16(atomic.AddUint32(&sequence, 1))

	pinger.payloadMu.RLock()
	defer pinger.payloadMu.RUnlock()

	// start measurement (tStop is set in the receiving end)
	req.init()

	// send request
	err := pinger.conn.WriteTo(dest, int(seq), pinger.payload)

	// enqueue in currently running requests
	pinger.mtx.Lock()
	pinger.requests[seq] = req
	pinger.mtx.Unlock()

	// send failed, need to remove request from list
	if err != nil {
		req.close()
		pinger.removeRequest(seq)

		return 0, err
	}

	return seq, nil
}
