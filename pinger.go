package ping

import (
	"os"
	"sync"
	"time"

	"golang.org/x/net/icmp"
)

const (
	// ProtocolICMP is the number of the Internet Control Message Protocol
	ProtocolICMP = 1

	// ProtocolICMPv6 is the IPv6 Next Header value for ICMPv6
	ProtocolICMPv6 = 58
)

// sequence number for this process
var sequence uint32

// Pinger is a instance for ICMP echo requests
type Pinger struct {
	Attempts uint          // number of attempts
	Timeout  time.Duration // timeout per request

	requests map[uint16]*request // currently running requests
	mtx      sync.RWMutex        // lock for the requests map
	id       uint16
	conn4    *icmp.PacketConn
	conn6    *icmp.PacketConn
	wg       sync.WaitGroup
}

// New creates a new Pinger. This will open the raw socket and start the
// receiving logic. You'll need to call Close() to cleanup.
func New(bind4, bind6 string) (*Pinger, error) {
	// open sockets
	conn4, err := connectICMP("ip4:icmp", bind4)
	if err != nil {
		return nil, err
	}

	conn6, err := connectICMP("ip6:ipv6-icmp", bind6)
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
		conn4:    conn4,
		conn6:    conn6,
		id:       uint16(os.Getpid()),
		requests: make(map[uint16]*request),
	}

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

// connectICMP opens a new ICMP connection, iff network and address are not emtpy.
func connectICMP(network, address string) (*icmp.PacketConn, error) {
	if network == "" || address == "" {
		return nil, nil
	}

	return icmp.ListenPacket(network, address)
}

func (pinger *Pinger) close(conn *icmp.PacketConn) {
	if conn != nil {
		conn.Close()
	}
}
