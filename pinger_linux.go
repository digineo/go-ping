package ping

import (
	"errors"
	"os"
	"reflect"
	"syscall"

	"golang.org/x/net/icmp"
)

// getFD gets the system file descriptor for an icmp.PacketConn
func getFD(c *icmp.PacketConn) (uintptr, error) {
	v := reflect.ValueOf(c).Elem().FieldByName("c").Elem()
	if v.Elem().Kind() != reflect.Struct {
		return 0, errors.New("invalid type")
	}

	fd := v.Elem().FieldByName("conn").FieldByName("fd")
	if fd.Elem().Kind() != reflect.Struct {
		return 0, errors.New("invalid type")
	}

	pfd := fd.Elem().FieldByName("pfd")
	if pfd.Kind() != reflect.Struct {
		return 0, errors.New("invalid type")
	}

	return uintptr(pfd.FieldByName("Sysfd").Int()), nil
}

func (pinger *Pinger) SetMark(mark uint) error {
	conn4, ok := pinger.conn4.(*icmp.PacketConn)
	if !ok {
		return errors.New("invalid connection type")
	}

	fd, err := getFD(conn4)
	if err != nil {
		return err
	}

	err = os.NewSyscallError(
		"setsockopt",
		syscall.SetsockoptInt(int(fd), syscall.SOL_SOCKET, syscall.SO_MARK, int(mark)),
	)

	if err != nil {
		return err
	}

	conn6, ok := pinger.conn6.(*icmp.PacketConn)
	if !ok {
		return errors.New("invalid connection type")
	}

	fd, err = getFD(conn6)
	if err != nil {
		return err
	}

	return os.NewSyscallError(
		"setsockopt",
		syscall.SetsockoptInt(int(fd), syscall.SOL_SOCKET, syscall.SO_MARK, int(mark)),
	)
}
