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
			pinger, err := NewUDP(test.bind4, test.bind6)
			if err != nil && os.Getenv("RUN_UNPRIVILEGED") != "1" &&
				(errors.Is(err, syscall.EACCES) || errors.Is(err, syscall.EPERM)) {
				t.Skipf("Unprivileged ICMP is not permitted: %v", err)
			}
			require.NoError(t, err)
			t.Cleanup(pinger.Close)
			pingLoopback(t, pinger, test.targets)
		})
	}
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
