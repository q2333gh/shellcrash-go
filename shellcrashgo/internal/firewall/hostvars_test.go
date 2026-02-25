package firewall

import "testing"

func TestParseLocalIPv4(t *testing.T) {
	route := `
default via 192.168.1.1 dev eth0 proto dhcp src 192.168.1.2 metric 100
10.0.0.0/24 dev iot0 proto kernel scope link src 10.0.0.1
172.18.0.0/16 dev docker0 proto kernel scope link src 172.18.0.1
192.168.1.0/24 dev eth0 proto kernel scope link src 192.168.1.2
`
	got := parseLocalIPv4(route, true)
	if got != "192.168.1.2" {
		t.Fatalf("filtered parseLocalIPv4 mismatch, got %q", got)
	}

	got = parseLocalIPv4(route, false)
	if got != "192.168.1.2 10.0.0.1 172.18.0.1" {
		t.Fatalf("unfiltered parseLocalIPv4 mismatch, got %q", got)
	}
}
