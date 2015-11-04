package pingers

import (
	"bytes"
	"golang.org/x/net/icmp"
	"golang.org/x/net/ipv4"
	"log"
	"net"
	"net/url"
	"os"
	"sync"
	"time"
)

func init() {
	pingers["icmp"] = pingerICMP
}

var (
	icmpSequence      uint16
	icmpSequenceMutex sync.Mutex
)

func getICMPSequence() uint16 {
	icmpSequenceMutex.Lock()
	defer icmpSequenceMutex.Unlock()
	icmpSequence += 1
	return icmpSequence
}

func pingerICMP(url *url.URL, m Metrics) {

	target := url.Host

	deadline := time.Now().Add(*timeout)
	m.Up.WithLabelValues(url.String()).Set(0)

	socket, err := icmp.ListenPacket("udp4", "0.0.0.0")
	if err != nil {
		log.Printf("Error listening to socket: %s", err)
		return
	}
	defer socket.Close()

	start := time.Now()
	defer m.Latency.WithLabelValues(url.String()).Set(time.Since(start).Seconds())

	ip, err := net.ResolveIPAddr("ip4", target)
	if err != nil {
		log.Printf("Error resolving address %s: %s", target, err)
		return
	}

	seq := getICMPSequence()
	pid := os.Getpid() & 0xffff

	wm := icmp.Message{
		Type: ipv4.ICMPTypeEcho, Code: 0,
		Body: &icmp.Echo{
			ID: pid, Seq: int(seq),
			Data: []byte("blackbox_prober"),
		},
	}
	wb, err := wm.Marshal(nil)
	if err != nil {
		log.Printf("Error marshalling packet for %s: %s", target, err)
		return
	}
	if _, err := socket.WriteTo(wb, ip); err != nil {
		log.Printf("Error writing to socket for %s: %s", target, err)
		return
	}

	// Reply should be the same except for the message type.
	wm.Type = ipv4.ICMPTypeEchoReply
	wb, err = wm.Marshal(nil)
	if err != nil {
		log.Printf("Error marshalling packet for %s: %s", target, err)
		return
	}

	rb := make([]byte, 1500)
	if err := socket.SetReadDeadline(deadline); err != nil {
		log.Printf("Error setting socket deadline for %s: %s", target, err)
		return
	}
	for {
		n, peer, err := socket.ReadFrom(rb)
		if err != nil {
			if nerr, ok := err.(net.Error); ok && nerr.Timeout() {
				log.Printf("Timeout reading from socket for %s: %s", target, err)
				return
			}
			log.Printf("Error reading from socket for %s: %s", target, err)
			continue
		}
		if peer.String() != ip.String() {
			continue
		}
		if bytes.Compare(rb[:n], wb) == 0 {
			m.Up.WithLabelValues(url.String()).Set(1)
			return
		}
	}
	m.Up.WithLabelValues(url.String()).Set(1)
	return
}
