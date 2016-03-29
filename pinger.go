package ping

import (
	"fmt"
	"golang.org/x/net/icmp"
	"golang.org/x/net/ipv4"
	"net"
	"os"
	"time"
)

const (
	ProtocolICMP     = 1  // Internet Control Message
	ProtocolIPv6ICMP = 58 // ICMP for IPv6
)

type Pinger struct {
	Local    net.IP
	Remote   net.IPAddr
	Attempts int           // Anzahl der Versuche
	Timeout  time.Duration // Timeout pro Ping
}

// Sendet Pings bis einer erfolgreich ist, oder die Versuche ausgeschöpft sind
func (pinger *Pinger) Ping() error {
	// Verbindung instanziieren
	c, err := icmp.ListenPacket("ip4:icmp", pinger.Local.String())
	if err != nil {
		return err
	}
	defer c.Close()

	// Mehrere Versuche
	for i := 0; i < pinger.Attempts; i++ {
		// Timeout setzen
		c.SetDeadline(time.Now().Add(pinger.Timeout))

		// Pingen
		if err = pinger.once(c, i+1); err == nil {
			// erfolgreich
			break
		}
	}

	return err
}

// Schickt einen Ping ab und wartet auf Antwort
func (pinger *Pinger) once(c *icmp.PacketConn, seq int) error {

	// Paket bauen
	wm := icmp.Message{
		Type: ipv4.ICMPTypeEcho, Code: 0,
		Body: &icmp.Echo{
			ID:   os.Getpid() & 0xffff,
			Seq:  seq,
			Data: []byte("HELLO-R-U-THERE"),
		},
	}

	// Paket serialisieren
	wb, err := wm.Marshal(nil)
	if err != nil {
		return err
	}

	// Paket abschicken
	if _, err := c.WriteTo(wb, &pinger.Remote); err != nil {
		return err
	}

	// Antwort einlesen
	rb := make([]byte, 1500)
	n, _, err := c.ReadFrom(rb)

	if err != nil {
		// z.B. Timeout
		return err
	}

	// Antwort parsen
	rm, err := icmp.ParseMessage(ProtocolICMP, rb[:n])
	if err != nil {
		return err
	}

	// Antwort auswerten
	switch rm.Type {
	case ipv4.ICMPTypeEchoReply:
		// erfolgreich
		return nil
	default:
		// Fehler
		return fmt.Errorf("got %+v; want echo reply", rm)
	}
}
