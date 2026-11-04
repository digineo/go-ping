package ping

import (
	"errors"
	"os"
	"syscall"
)

// SetMark sets the SO_MARK socket option (fwmark) on all bound sockets.
func (pinger *Pinger) SetMark(mark uint) error {
	var conns []syscall.Conn

	if pinger.conn4 != nil {
		conns = append(conns, pinger.conn4.IPv4PacketConn().PacketConn.(syscall.Conn))
	}
	if pinger.conn6 != nil {
		conns = append(conns, pinger.conn6.IPv6PacketConn().PacketConn.(syscall.Conn))
	}

	for _, conn := range conns {
		if err := setMark(conn, int(mark)); err != nil {
			return err
		}
	}
	return nil
}

func setMark(conn syscall.Conn, mark int) error {
	raw, err := conn.SyscallConn()
	if err != nil {
		return err
	}

	var sockErr error
	err = raw.Control(func(fd uintptr) {
		sockErr = syscall.SetsockoptInt(int(fd), syscall.SOL_SOCKET, syscall.SO_MARK, mark)
	})
	return errors.Join(err, os.NewSyscallError("setsockopt", sockErr))
}
