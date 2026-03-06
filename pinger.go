package ping

import (
	"net"
	"os"
	"sync"

	"golang.org/x/net/icmp"
)

const (
	// ProtocolICMP is the number of the Internet Control Message Protocol
	// (see golang.org/x/net/internal/iana.ProtocolICMP)
	ProtocolICMP = 1

	// ProtocolICMPv6 is the IPv6 Next Header value for ICMPv6
	// see golang.org/x/net/internal/iana.ProtocolIPv6ICMP
	ProtocolICMPv6 = 58
)

// default sequence counter for this process
var sequence uint32

// Pinger is a instance for ICMP echo requests
type Pinger struct {
	LogUnexpectedPackets bool // increases log verbosity
	Id                   uint16
	SequenceCounter      *uint32

	payload   Payload
	payloadMu sync.RWMutex

	requests map[uint32]request // currently running requests
	mtx      sync.RWMutex       // lock for the requests map
	conn4    net.PacketConn
	conn6    net.PacketConn
	write4   sync.Mutex // lock for conn4.WriteTo
	write6   sync.Mutex // lock for conn6.WriteTo
	wg       sync.WaitGroup
}

// New creates a new Pinger. This will open the raw socket and start the
// receiving logic. You'll need to call Close() to cleanup.
func New(bind4, bind6 string) (*Pinger, error) {
	return NewPinger("ip4:icmp", bind4, "ip6:ipv6-icmp", bind6)
}

// NewUDP creates a new Pinger using unprivileged UDP sockets.
// Currently only Darwin and Linux support this.
func NewUDP(bind4, bind6 string) (*Pinger, error) {
	return NewPinger("udp4", bind4, "udp6", bind6)
}

// NewPinger creates a new Pinger with the given network types and bind addresses.
// For non-privileged ICMP, use "udp4" and "udp6".
func NewPinger(network4, bind4, network6, bind6 string) (*Pinger, error) {
	// open sockets
	conn4, err := connectICMP(network4, bind4)
	if err != nil {
		return nil, err
	}

	conn6, err := connectICMP(network6, bind6)
	if err != nil {
		if conn4 != nil {
			conn4.Close()
		}
		return nil, err
	}

	if conn4 == nil && conn6 == nil {
		return nil, errNotBound
	}

	pinger := Pinger{
		conn4:           conn4,
		conn6:           conn6,
		Id:              uint16(os.Getpid()),
		SequenceCounter: &sequence,
		requests:        make(map[uint32]request),
	}

	// For unprivileged sockets, we must use the port number as ID
	if conn4 != nil {
		if addr, ok := conn4.LocalAddr().(*net.UDPAddr); ok {
			pinger.Id = uint16(addr.Port)
		}
	} else if conn6 != nil {
		if addr, ok := conn6.LocalAddr().(*net.UDPAddr); ok {
			pinger.Id = uint16(addr.Port)
		}
	}

	pinger.SetPayloadSize(56)

	if conn4 != nil {
		pinger.wg.Add(1)
		go pinger.receiver(ProtocolICMP, pinger.conn4)
	}
	if conn6 != nil {
		pinger.wg.Add(1)
		go pinger.receiver(ProtocolICMPv6, pinger.conn6)
	}

	return &pinger, nil
}

// Close will close the ICMP socket.
func (pinger *Pinger) Close() {
	pinger.close(pinger.conn4)
	pinger.close(pinger.conn6)
	pinger.wg.Wait()
}

// connectICMP opens a new ICMP connection, if network and address are not empty.
func connectICMP(network, address string) (*icmp.PacketConn, error) {
	if network == "" || address == "" {
		return nil, nil
	}

	return icmp.ListenPacket(network, address)
}

func (pinger *Pinger) close(conn net.PacketConn) {
	if conn != nil {
		conn.Close()
	}
}

func (pinger *Pinger) removeRequest(idseq uint32) {
	pinger.mtx.Lock()
	delete(pinger.requests, idseq)
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
	pinger.payload = Payload(data)
	pinger.payloadMu.Unlock()
}

// PayloadSize retrieves the current payload size.
func (pinger *Pinger) PayloadSize() uint16 {
	pinger.payloadMu.RLock()
	defer pinger.payloadMu.RUnlock()
	return uint16(len(pinger.payload))
}
