package ping

import (
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"golang.org/x/net/icmp"
	"golang.org/x/net/ipv4"
	"golang.org/x/net/ipv6"
)

// replyPacketConn returns one reply, then ends the receiver loop.
type replyPacketConn struct {
	net.PacketConn
	packet []byte
	source net.Addr
}

func (conn *replyPacketConn) ReadFrom(buf []byte) (int, net.Addr, error) {
	if conn.packet == nil {
		return 0, nil, net.ErrClosed
	}
	n := copy(buf, conn.packet)
	conn.packet = nil
	return n, conn.source, nil
}

func TestReceiverSourceAddress(t *testing.T) {
	for _, test := range []struct {
		name      string
		proto     int
		replyType icmp.Type
		ip        string
	}{
		{"IPv4", ProtocolICMP, ipv4.ICMPTypeEchoReply, "127.0.0.1"},
		{"IPv6", ProtocolICMPv6, ipv6.ICMPTypeEchoReply, "::1"},
	} {
		t.Run(test.name, func(t *testing.T) {
			ip := net.ParseIP(test.ip)
			for _, source := range []net.Addr{
				&net.IPAddr{IP: ip},
				&net.UDPAddr{IP: ip, Port: 1234},
			} {
				t.Run(source.Network(), func(t *testing.T) {
					message := icmp.Message{
						Type: test.replyType,
						Body: &icmp.Echo{ID: 1234, Seq: 1},
					}
					packet, err := message.Marshal(nil)
					require.NoError(t, err)

					req := &multiRequest{}
					req.init()
					t.Cleanup(req.close)
					pinger := &Pinger{
						requests: map[uint32]request{makeIDSeq(1234, 1): req},
					}
					pinger.wg.Add(1)
					pinger.receiver(test.proto, &replyPacketConn{packet: packet, source: source})

					select {
					case reply := <-req.replies:
						require.True(t, ip.Equal(reply.Address), "reply source = %v, want %v", reply.Address, ip)
					case <-time.After(time.Second):
						t.Fatal("receiver did not deliver the echo reply")
					}
				})
			}
		})
	}
}
