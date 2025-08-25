//go:build !linux

package ping

import "errors"

func (pinger *Pinger) SetMark(mark uint) error {
	return errors.New("setting SO_MARK socket option is not supported on this platform")
}
