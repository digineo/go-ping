package ping

import (
	"net"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSendRequestWithoutConnection(t *testing.T) {
	for _, target := range []string{"127.0.0.1", "::1"} {
		t.Run(target, func(t *testing.T) {
			var sequence uint32 = 42
			pinger := &Pinger{SequenceCounter: &sequence}
			req := &simpleRequest{}

			idseq, err := pinger.sendRequest(&net.IPAddr{IP: net.ParseIP(target)}, req)

			require.ErrorIs(t, err, errNoConn)
			require.Zero(t, idseq)
			require.Equal(t, uint32(42), sequence, "failed requests must not consume a sequence number")
			require.Nil(t, req.wait, "failed requests must not be initialized")
			require.Empty(t, pinger.requests)
		})
	}
}
