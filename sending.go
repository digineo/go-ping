package ping

import (
	"context"
	"errors"
	"net"
	"sync"
	"sync/atomic"
	"time"

	"golang.org/x/net/icmp"
	"golang.org/x/net/ipv4"
	"golang.org/x/net/ipv6"
)

// PingAttempts sends ICMP echo requests with a timeout per request, retrying upto `attempt` times .
// Will finish early on success and return the round trip time of the last ping.
func (pinger *Pinger) PingAttempts(remote *net.IPAddr, timeout time.Duration, attempts int) (rtt time.Duration, err error) {
	if attempts < 1 {
		err = errors.New("zero attempts")
	} else {
		for i := 0; i < attempts; i++ {
			rtt, err = pinger.Ping(remote, timeout)

			if err == nil {
				break // success
			}
		}
	}
	return
}

// Ping sends a single Echo Request and waits for an answer. It returns
// the round trip time (RTT) if a reply is received in time.
func (pinger *Pinger) Ping(remote *net.IPAddr, timeout time.Duration) (time.Duration, error) {
	ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(timeout))
	defer cancel()
	return pinger.PingContext(ctx, remote)
}

// PingContext sends a single Echo Request and waits for an answer. It returns
// the round trip time (RTT) if a reply is received before cancellation of the context.
func (pinger *Pinger) PingContext(ctx context.Context, remote *net.IPAddr) (time.Duration, error) {
	req := simpleRequest{}
	seq, err := pinger.sendRequest(remote, &req)

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

// PingMulticast sends a single Echo Request and waits a given time for the answer(s)
func (pinger *Pinger) PingMulticast(ctx context.Context, remote *net.IPAddr) (<-chan Response, error) {

	req := multiRequest{
		responses: make(chan Response),
	}
	seq, err := pinger.sendRequest(remote, &req)

	if err != nil {
		return nil, err
	}

	go func() {
		<-ctx.Done()

		// dequeue request
		pinger.removeRequest(seq)

		req.close()
	}()

	return req.responses, nil
}

func (pinger *Pinger) sendRequest(remote *net.IPAddr, req request) (uint16, error) {
	seq := uint16(atomic.AddUint32(&sequence, 1))

	pinger.payloadMu.RLock()
	defer pinger.payloadMu.RUnlock()

	// build packet
	wm := icmp.Message{
		Code: 0,
		Body: &icmp.Echo{
			ID:   int(pinger.id),
			Seq:  int(seq),
			Data: pinger.payload,
		},
	}

	// Protocol specifics
	var conn *icmp.PacketConn
	var lock *sync.Mutex
	if remote.IP.To4() != nil {
		wm.Type = ipv4.ICMPTypeEcho
		conn = pinger.conn4
		lock = &pinger.write4
	} else {
		wm.Type = ipv6.ICMPTypeEchoRequest
		conn = pinger.conn6
		lock = &pinger.write6
	}

	// serialize packet
	wb, err := wm.Marshal(nil)
	if err != nil {
		return seq, err
	}

	// enqueue in currently running requests
	pinger.mtx.Lock()
	pinger.requests[seq] = req
	pinger.mtx.Unlock()

	// start measurement (tStop is set in the receiving end)
	lock.Lock()
	req.init()

	// send request
	_, err = conn.WriteTo(wb, remote)
	lock.Unlock()

	// send failed, need to remove request from list
	if err != nil {
		req.close()
		pinger.removeRequest(seq)

		return 0, err
	}

	return seq, nil
}
