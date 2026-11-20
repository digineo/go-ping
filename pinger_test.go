package ping

import (
	"errors"
	"net"
	"os"
	"runtime"
	"syscall"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestPinger_NoBindings(t *testing.T) {
	for _, test := range []struct {
		name string
		new  func(string, string) (*Pinger, error)
	}{
		{"Privileged", New},
		{"Unprivileged", NewUDP},
	} {
		t.Run(test.name, func(t *testing.T) {
			pinger, err := test.new("", "")
			require.ErrorIs(t, err, errNotBound)
			require.Nil(t, pinger)
		})
	}
}

func TestPinger_Privileged(t *testing.T) {
	if os.Getenv("RUN_PRIVILEGED") != "1" {
		t.Skip("Run with RUN_PRIVILEGED=1 and administrative rights.")
	}

	pinger, err := New("0.0.0.0", "::")
	require.NoError(t, err)
	t.Cleanup(pinger.Close)
	// Privileged sockets must continue to honor the configurable identifier.
	pinger.Id = 0x1234
	pingLoopback(t, pinger, []string{"127.0.0.1", "::1"})
}

func TestPinger_Unprivileged(t *testing.T) {
	if runtime.GOOS != "linux" && runtime.GOOS != "darwin" {
		t.Skip("Unprivileged ICMP sockets require Linux or Darwin.")
	}

	for _, test := range []struct {
		name    string
		bind4   string
		bind6   string
		targets []string
	}{
		{"IPv4", "0.0.0.0", "", []string{"127.0.0.1"}},
		{"IPv6", "", "::", []string{"::1"}},
		{"DualStack", "0.0.0.0", "::", []string{"127.0.0.1", "::1"}},
	} {
		t.Run(test.name, func(t *testing.T) {
			pinger := newUDP(t, test.bind4, test.bind6)
			pingLoopback(t, pinger, test.targets)
		})
	}
}

// Two processes using unprivileged sockets must not accept each other's
// replies. Darwin delivers every echo reply to all ICMP sockets.
func TestPinger_UnprivilegedIgnoresForeignReplies(t *testing.T) {
	if runtime.GOOS != "linux" && runtime.GOOS != "darwin" {
		t.Skip("Unprivileged ICMP sockets require Linux or Darwin.")
	}

	// simulate two processes with their own identifiers and sequence counters
	var seqA, seqB uint32
	a := newUDP(t, "0.0.0.0", "")
	a.Id, a.SequenceCounter = 1, &seqA
	b := newUDP(t, "0.0.0.0", "")
	b.Id, b.SequenceCounter = 2, &seqB

	// TEST-NET-1 (RFC 5737) does not answer
	replies, err := b.PingMulticast(&net.IPAddr{IP: net.ParseIP("192.0.2.1")}, 500*time.Millisecond)
	require.NoError(t, err)

	_, err = a.Ping(&net.IPAddr{IP: net.ParseIP("127.0.0.1")}, time.Second)
	require.NoError(t, err)

	for reply := range replies {
		t.Errorf("unexpected reply from %v", reply.Address)
	}
}

// newUDP opens an unprivileged pinger, skipping the test if the system does
// not permit it and RUN_UNPRIVILEGED is not set.
func newUDP(t *testing.T, bind4, bind6 string) *Pinger {
	t.Helper()
	pinger, err := NewUDP(bind4, bind6)
	if err != nil && os.Getenv("RUN_UNPRIVILEGED") != "1" &&
		(errors.Is(err, syscall.EACCES) || errors.Is(err, syscall.EPERM)) {
		t.Skipf("Unprivileged ICMP is not permitted: %v", err)
	}
	require.NoError(t, err)
	t.Cleanup(pinger.Close)
	return pinger
}

func pingLoopback(t *testing.T, pinger *Pinger, targets []string) {
	t.Helper()
	for _, target := range targets {
		t.Run(target, func(t *testing.T) {
			rtt, err := pinger.PingAttempts(&net.IPAddr{IP: net.ParseIP(target)}, time.Second, 2)
			require.NoError(t, err)
			require.Positive(t, rtt)
		})
	}
}
