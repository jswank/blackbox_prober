package pingers

import (
	"golang.org/x/net/icmp"
	"net/url"
	"testing"

	dto "github.com/prometheus/client_model/go"
)

func TestICMP(t *testing.T) {

	socket, err := icmp.ListenPacket("udp4", "0.0.0.0")
	if err != nil {
		t.Log("ICMP echo/reply is non-functional for this user")
		return
	}
	defer socket.Close()

	u, err := url.Parse("icmp://localhost")
	if err != nil {
		t.Fatal(err)
	}
	pingerICMP(u, metrics)

	pb := &dto.Metric{}
	metrics.Up.WithLabelValues(u.String()).Write(pb)
	if expected, got := 1., pb.Gauge.GetValue(); expected != got {
		t.Fatalf("Expected: %f, Got: %f", expected, got)
	}

	metrics.Latency.WithLabelValues(u.String()).Write(pb)
	if pb.Gauge.GetValue() == 0 {
		t.Fatal("Expected non-zero value")
	}
}
