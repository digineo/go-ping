package ping

import (
	"net"
	"sync/atomic"
	"time"

	"golang.org/x/net/icmp"
	"golang.org/x/net/ipv4"
)

// Ping sends ICMP echo requests, retrying upto Pinger.Attempts times.
// Will finishes early on success.
func (pinger *Pinger) Ping(remote net.Addr) (err error) {
	// multiple attempts
	for i := uint(0); i < pinger.Attempts; i++ {
		// set timeout
		pinger.conn.SetDeadline(time.Now().Add(pinger.Timeout))

		if err = pinger.once(remote); err == nil {
			break // success
		}
	}

	return
}

// once sends a single Echo Request and waits for an answer.
func (pinger *Pinger) once(remote net.Addr) error {
	seq := uint16(atomic.AddUint32(&sequence, 1))
	req := request{
		wait: make(chan struct{}),
	}

	// build packet
	wm := icmp.Message{
		Type: ipv4.ICMPTypeEcho,
		Code: 0,
		Body: &icmp.Echo{
			ID:  int(pinger.id),
			Seq: int(seq),
		},
	}
	// serialize packet
	wb, err := wm.Marshal(nil)
	if err != nil {
		return err
	}

	// enqueue in currently running requests
	pinger.mtx.Lock()
	pinger.requests[seq] = &req
	pinger.mtx.Unlock()

	// send request
	if _, e := pinger.conn.WriteTo(wb, remote); e != nil {
		req.respond(e)
	}

	// wait for answer
	select {
	case <-req.wait:
		err = req.result
	case <-time.After(pinger.Timeout):
		err = &timeoutError{}
	}

	// dequeue request
	pinger.mtx.Lock()
	delete(pinger.requests, seq)
	pinger.mtx.Unlock()

	return err
}
