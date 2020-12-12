package ping

import (
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPinger(t *testing.T) {
	assert := assert.New(t)
	require := require.New(t)

	pinger, err := New("0.0.0.0", "::", false)
	require.NoError(err)
	require.NotNil(pinger)
	defer pinger.Close()

	for _, target := range []string{"127.0.0.1", "::1"} {
		rtt, err := pinger.PingAttempts(&net.IPAddr{IP: net.ParseIP(target)}, time.Second, 2)
		assert.NoError(err, target)
		assert.NotZero(rtt, target)
	}
}
