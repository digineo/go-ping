package ping

import (
	"fmt"
	"net"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPinger_Privileged(t *testing.T) {
	if os.Getenv("RUN_PRIVILEGED") != "1" {
		t.Skip("Skipping privileged test. Run with RUN_PRIVILEGED=1 and administrative rights.")
	}
	assert := assert.New(t)
	require := require.New(t)

	pinger, err := New("0.0.0.0", "::")
	require.NoError(err)
	require.NotNil(pinger)
	defer pinger.Close()

	for _, target := range []string{"8.8.8.8"} {
		fmt.Printf("Pinging (privileged) %s...\n", target)
		rtt, err := pinger.PingAttempts(&net.IPAddr{IP: net.ParseIP(target)}, time.Second, 2)
		assert.NoError(err, target)
		assert.True(rtt >= 0, target)
	}
}

func TestPinger_Unprivileged(t *testing.T) {
	// Note: Windows doesn't actually support unprivileged ICMP via "udp4",
	// but we can still test if our code initializes and handles the types.
	// This will likely fail to bind on Windows, so we check the error.
	assert := assert.New(t)
	require := require.New(t)

	pinger, err := NewUDP("0.0.0.0", "::")
	if err != nil {
		t.Skipf("Skipping unprivileged test (likely not supported on this OS or environment): %v", err)
		return
	}
	require.NotNil(pinger)
	defer pinger.Close()

	for _, target := range []string{"8.8.8.8"} {
		fmt.Printf("Pinging (unprivileged) %s...\n", target)
		rtt, err := pinger.PingAttempts(&net.IPAddr{IP: net.ParseIP(target)}, time.Second, 2)
		assert.NoError(err, target)
		assert.True(rtt >= 0, target)
	}
}

func TestIDHandling(t *testing.T) {
	// Test if NewPinger correctly picks up the port as ID when using UDP
	p, err := NewPinger("udp4", "0.0.0.0", "", "") // Bind to 0.0.0.0
	if err != nil {
		t.Skipf("Could not bind UDP socket for ID test: %v", err)
		return
	}
	defer p.Close()

	require.NotEqual(t, uint16(os.Getpid()), p.Id, "In UDP mode, ID should be the port, not PID")
	require.NotZero(t, p.Id, "Port-based ID should not be zero")
	fmt.Printf("Pinger ID (port): %d, PID was: %d\n", p.Id, os.Getpid())
}
