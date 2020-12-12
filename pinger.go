package ping

import (
	"sync"

	"github.com/digineo/go-ping/internal"
)

const (
	// ProtocolICMP is the number of the Internet Control Message Protocol
	// (see golang.org/x/net/internal/iana.ProtocolICMP)
	ProtocolICMP = 1

	// ProtocolICMPv6 is the IPv6 Next Header value for ICMPv6
	// see golang.org/x/net/internal/iana.ProtocolIPv6ICMP
	ProtocolICMPv6 = 58
)

// Pinger is a instance for ICMP echo requests
type Pinger struct {
	LogUnexpectedPackets bool // increases log verbosity

	payload   internal.Payload
	payloadMu sync.RWMutex

	requests map[uint16]request // currently running requests
	mtx      sync.RWMutex       // lock for the requests map
	conn     internal.Conn

	wg sync.WaitGroup
}

// New creates a new Pinger. This will open the raw socket and start the
// receiving logic. You'll need to call Close() to cleanup.
func New(bind4, bind6 string, privileged bool) (*Pinger, error) {
	pinger := Pinger{}
	pinger.conn.Privileged = privileged

	err := pinger.conn.Open(bind4, bind6)
	if err != nil {
		return nil, err
	}

	pinger.requests = make(map[uint16]request)
	pinger.SetPayloadSize(56)
	pinger.conn.Receiver = pinger.process

	return &pinger, nil
}

// Close will close the ICMP socket.
func (pinger *Pinger) Close() {
	pinger.conn.Close()
	pinger.wg.Wait()
}

func (pinger *Pinger) removeRequest(seq uint16) {
	pinger.mtx.Lock()
	delete(pinger.requests, seq)
	pinger.mtx.Unlock()
}

// SetPayloadSize resizes additional payload data to the given size. The
// payload will subsequently be appended to outgoing ICMP Echo Requests.
//
// The default payload size is 56, resulting in 64 bytes for the ICMP packet.
func (pinger *Pinger) SetPayloadSize(size uint16) {
	pinger.payloadMu.Lock()
	pinger.payload.Resize(size)
	pinger.payloadMu.Unlock()
}

// SetPayload allows you to overwrite the current payload with your own data.
func (pinger *Pinger) SetPayload(data []byte) {
	pinger.payloadMu.Lock()
	pinger.payload = internal.Payload(data)
	pinger.payloadMu.Unlock()
}

// PayloadSize retrieves the current payload size.
func (pinger *Pinger) PayloadSize() uint16 {
	pinger.payloadMu.RLock()
	defer pinger.payloadMu.RUnlock()
	return uint16(len(pinger.payload))
}
