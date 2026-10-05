// Package discovery publishes stable per-installation desktop LAN addresses.
package discovery

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"github.com/libp2p/zeroconf/v2"
	"log"
	"net"
	"runtime"
	"sort"
	"strings"
	"sync"
	"time"
)

type Status struct {
	DeviceID string `json:"device_id"`
	Hostname string `json:"hostname"`
	Port     int    `json:"port"`
	State    string `json:"state"`
}
type Publisher struct {
	mu     sync.RWMutex
	status Status
	host   string
}

func New(id, host string, port int) *Publisher {
	prefix := map[string]string{"windows": "win", "darwin": "mac"}[runtime.GOOS]
	status := Status{Port: port, State: "disabled"}
	if prefix != "" {
		status.DeviceID = id
		status.Hostname = "questrace-" + prefix + "-" + id + ".local"
		status.State = "starting"
	}
	return &Publisher{status: status, host: host}
}
func (p *Publisher) Status() Status    { p.mu.RLock(); defer p.mu.RUnlock(); return p.status }
func (p *Publisher) setState(s string) { p.mu.Lock(); p.status.State = s; p.mu.Unlock() }
func (p *Publisher) URL() string {
	s := p.Status()
	if s.State != "available" {
		return ""
	}
	return fmt.Sprintf("http://%s:%d", s.Hostname, s.Port)
}

// virtualAdapterMarkers identify adapters that hold a private address but cannot
// be reached from a phone on the same Wi-Fi: hypervisor switches (Hyper-V, WSL,
// VMware, VirtualBox), container bridges, VPN and tunnel interfaces, and the
// Wi-Fi Direct / hotspot adapters Windows names "Local Area Connection* N".
// Adapter names are localized, so only product names and stable fragments are
// listed. Matching is case-insensitive substring matching; the trailing "*" in
// one marker is a literal character of the Windows name, not a wildcard.
var virtualAdapterMarkers = []string{
	// Windows
	"vethernet", "hyper-v", "wsl", "vmware", "virtualbox", "vbox",
	"local area connection*", "wi-fi direct", "microsoft wi-fi direct",
	"bluetooth", "loopback", "teredo", "isatap", "npcap", "tap-windows",
	"openvpn", "wireguard", "tailscale", "zerotier",
	// macOS / iOS
	"utun", "awdl", "llw", "vmenet", "anpi", "ap1", "ipsec", "vmnet", "vboxnet",
	// Linux and containers
	"docker", "veth", "br-", "virbr", "podman", "cni", "tun", "tap", "wg", "zt",
}

// isVirtualAdapter reports whether an adapter name belongs to a virtual or
// tunnel adapter rather than a physical network the phone shares.
func isVirtualAdapter(name string) bool {
	lower := strings.ToLower(name)
	for _, marker := range virtualAdapterMarkers {
		if strings.Contains(lower, marker) {
			return true
		}
	}
	return false
}

// lanInterface is one interface together with the addresses it may advertise.
type lanInterface struct {
	nic net.Interface
	ips []string
}

// selectLANInterfaces prefers the adapters a peer can actually reach. When every
// candidate looks virtual the candidates are returned unchanged: a machine whose
// only path is a bridge or a tunnel must keep its discovery rather than lose it.
func selectLANInterfaces(candidates []lanInterface) []lanInterface {
	var preferred []lanInterface
	for _, candidate := range candidates {
		if !isVirtualAdapter(candidate.nic.Name) {
			preferred = append(preferred, candidate)
		}
	}
	if len(preferred) == 0 {
		return candidates
	}
	return preferred
}

func Interfaces(host string) ([]net.Interface, []string) {
	all, _ := net.Interfaces()
	var candidates []lanInterface
	for _, nic := range all {
		if nic.Flags&net.FlagUp == 0 || nic.Flags&net.FlagLoopback != 0 || nic.Flags&net.FlagMulticast == 0 {
			continue
		}
		addresses, _ := nic.Addrs()
		var found []string
		for _, a := range addresses {
			ip, _, err := net.ParseCIDR(a.String())
			if err != nil || ip.To4() == nil || !ip.IsPrivate() {
				continue
			}
			if host != "" && host != "0.0.0.0" && host != "::" && host != ip.String() {
				continue
			}
			found = append(found, ip.String())
		}
		if len(found) > 0 {
			candidates = append(candidates, lanInterface{nic: nic, ips: found})
		}
	}
	var ifaces []net.Interface
	var ips []string
	for _, candidate := range selectLANInterfaces(candidates) {
		ifaces = append(ifaces, candidate.nic)
		ips = append(ips, candidate.ips...)
	}
	sort.Strings(ips)
	return ifaces, ips
}
func NetworkKey() string {
	_, ips := Interfaces("")
	sum := sha256.Sum256([]byte(strings.Join(ips, ",")))
	return hex.EncodeToString(sum[:])
}
func (p *Publisher) Run(ctx context.Context) {
	s := p.Status()
	if s.State == "disabled" {
		return
	}
	var server *zeroconf.Server
	var hostnameServer *hostnameResponder
	var signature string
	defer func() {
		if server != nil {
			server.Shutdown()
		}
		if hostnameServer != nil {
			hostnameServer.Close()
		}
		p.setState("stopped")
	}()
	refresh := func() {
		ifaces, ips := Interfaces(p.host)
		parts := append([]string{}, ips...)
		for _, i := range ifaces {
			parts = append(parts, fmt.Sprintf("%d:%s", i.Index, i.Name))
		}
		next := strings.Join(parts, ",")
		if next == signature && server != nil {
			return
		}
		if server != nil {
			server.Shutdown()
			server = nil
		}
		if hostnameServer != nil {
			hostnameServer.Close()
			hostnameServer = nil
		}
		signature = next
		if len(ips) == 0 {
			p.setState("unavailable")
			return
		}
		var err error
		server, err = zeroconf.RegisterProxy("Questrace-"+s.DeviceID, "_questrace._tcp", "local.", s.Port, s.Hostname+".", ips, []string{"id=" + s.DeviceID, "version=1", "path=/"}, ifaces)
		if err != nil {
			p.setState("unavailable")
		} else {
			// The hostname responder is an enhancement on top of the DNS-SD
			// advertisement: it only makes the .local URL resolvable for peers
			// whose resolver asks mDNS for it. Peers discover this computer (and
			// its A records) through the records registered above, so a responder
			// failure must never take the advertisement down with it.
			if hostnameServer, err = startHostnameResponder(s.Hostname, ifaces); err != nil {
				log.Printf("discovery: hostname responder unavailable: %v", err)
				hostnameServer = nil
			}
			p.setState("available")
		}
	}
	refresh()
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			refresh()
		}
	}
}
