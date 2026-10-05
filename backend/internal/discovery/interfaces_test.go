package discovery

import (
	"net"
	"testing"
)

func TestIsVirtualAdapter(t *testing.T) {
	for _, tc := range []struct {
		name    string
		virtual bool
	}{
		{"vEthernet (Default Switch)", true},
		{"vEthernet (WSL (Hyper-V firewall))", true},
		{"Local Area Connection* 10", true},
		{"VMware Network Adapter VMnet8", true},
		{"VirtualBox Host-Only Network", true},
		{"Tailscale", true},
		{"WireGuard Tunnel", true},
		{"utun3", true},
		{"awdl0", true},
		{"docker0", true},
		{"veth1a2b3c", true},
		{"br-9f1d2c", true},
		{"Wi-Fi Direct", true},
		{"Ethernet", false},
		{"Ethernet 2", false},
		{"Wi-Fi", false},
		{"en0", false},
		{"en7", false},
		{"eth0", false},
		{"wlan0", false},
		{"Thunderbolt Bridge", false},
		{"以太网", false},
	} {
		if got := isVirtualAdapter(tc.name); got != tc.virtual {
			t.Fatalf("%q: got %v, want %v", tc.name, got, tc.virtual)
		}
	}
}

func TestSelectLANInterfaces(t *testing.T) {
	real := lanInterface{nic: net.Interface{Name: "Wi-Fi", Index: 7}, ips: []string{"192.168.1.104"}}
	second := lanInterface{nic: net.Interface{Name: "Ethernet", Index: 9}, ips: []string{"10.0.0.7"}}
	hyperv := lanInterface{nic: net.Interface{Name: "vEthernet (Default Switch)", Index: 21}, ips: []string{"172.20.0.1"}}
	hotspot := lanInterface{nic: net.Interface{Name: "Local Area Connection* 10", Index: 23}, ips: []string{"192.168.137.1"}}

	got := selectLANInterfaces([]lanInterface{hyperv, real, hotspot, second})
	if len(got) != 2 || got[0].nic.Name != "Wi-Fi" || got[1].nic.Name != "Ethernet" {
		t.Fatalf("physical adapters not preferred: %+v", got)
	}

	// A machine whose only path is a virtual adapter keeps discovery instead of
	// silently losing it.
	got = selectLANInterfaces([]lanInterface{hyperv, hotspot})
	if len(got) != 2 {
		t.Fatalf("virtual-only fallback dropped adapters: %+v", got)
	}
}
