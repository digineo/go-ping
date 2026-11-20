package ping

import (
	"bytes"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/require"
	"golang.org/x/net/icmp"
)

func TestSetLogger(t *testing.T) {
	t.Cleanup(func() { SetLogger(nil) })

	var buf bytes.Buffer
	custom := slog.New(slog.NewTextHandler(&buf, nil))
	pinger := &Pinger{LogUnexpectedPackets: true}

	SetLogger(custom)
	require.Same(t, custom, getLogger())

	pinger.process(&icmp.DstUnreach{}, nil, nil, nil)
	require.Contains(t, buf.String(), `msg="expected *icmp.Echo" body=`)

	SetLogger(nil)
	require.Same(t, slog.Default(), getLogger())
}
