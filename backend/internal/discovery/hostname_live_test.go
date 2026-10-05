package discovery

import (
	"crypto/rand"
	"encoding/hex"
	"net"
	"os"
	"testing"
	"time"

	"github.com/miekg/dns"
	"golang.org/x/net/ipv4"
)

// These tests exercise the responder over real sockets: they join the mDNS group
// themselves, ask a multicast A question and wait for the unicast answer the
// responder must send back to a non-5353 source port (RFC 6762 section 6.7).
// They need working multicast egress on an interface with a private IPv4
// address, so they only run when QUESTRACE_MDNS_LIVE=1. Run them on every
// platform that has a second device on the LAN (a phone resolving the URL is
// the real end-to-end check).

// liveQueryConn returns a socket that sends multicast questions out of nic.
func liveQueryConn(t *testing.T, nic net.Interface) *ipv4.PacketConn {
	t.Helper()
	udp, err := net.ListenUDP("udp4", &net.UDPAddr{IP: net.IPv4(224, 0, 0, 0), Port: 0})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = udp.Close() })
	conn := ipv4.NewPacketConn(udp)
	if err := conn.SetMulticastInterface(&nic); err != nil {
		t.Fatal(err)
	}
	if err := conn.SetMulticastLoopback(true); err != nil {
		t.Fatal(err)
	}
	return conn
}

// ask sends an A question for hostname and returns the first matching answer, or
// nil when no answer arrives before wait elapses.
func ask(t *testing.T, conn *ipv4.PacketConn, hostname string, wait time.Duration) *dns.A {
	t.Helper()
	query := new(dns.Msg)
	query.SetQuestion(dns.Fqdn(hostname), dns.TypeA)
	query.RecursionDesired = false
	packet, err := query.Pack()
	if err != nil {
		t.Fatal(err)
	}
	var sendErr error
	for attempt := 0; attempt < 5; attempt++ {
		if _, sendErr = conn.WriteTo(packet, nil, mdnsGroup); sendErr == nil {
			break
		}
		time.Sleep(200 * time.Millisecond)
	}
	if sendErr != nil {
		t.Skipf("multicast egress unavailable in this environment: %v", sendErr)
	}
	if err := conn.SetReadDeadline(time.Now().Add(wait)); err != nil {
		t.Fatal(err)
	}
	buffer := make([]byte, 9000)
	for {
		n, _, _, err := conn.ReadFrom(buffer)
		if err != nil {
			return nil
		}
		reply := new(dns.Msg)
		if reply.Unpack(buffer[:n]) != nil || !reply.Response {
			continue
		}
		for _, answer := range reply.Answer {
			if a, ok := answer.(*dns.A); ok && a.Hdr.Name == dns.Fqdn(hostname) {
				return a
			}
		}
	}
}

// TestHostnameResponderLive checks the positive and the negative case: the
// advertised hostname is answered with a private address, and a name nobody
// registered is not answered at all (which shows the answer really comes from
// this responder and not from the platform resolver).
func TestHostnameResponderLive(t *testing.T) {
	if os.Getenv("QUESTRACE_MDNS_LIVE") != "1" {
		t.Skip("set QUESTRACE_MDNS_LIVE=1 to run the live mDNS tests")
	}
	ifaces, _ := Interfaces("")
	if len(ifaces) == 0 {
		t.Fatal("no interface with a private IPv4 address")
	}
	hostname := "questrace-test-hostname.local"
	responder, err := startHostnameResponder(hostname, ifaces)
	if err != nil {
		t.Fatalf("startHostnameResponder: %v", err)
	}
	defer responder.Close()

	conn := liveQueryConn(t, ifaces[0])
	a := ask(t, conn, hostname, 5*time.Second)
	if a == nil {
		t.Fatalf("no answer for %s", hostname)
	}
	if !a.A.IsPrivate() {
		t.Fatalf("answered a non-private address %s", a.A)
	}
	t.Logf("answered %s -> %s", hostname, a.A)

	unknown := make([]byte, 6)
	if _, err := rand.Read(unknown); err != nil {
		t.Fatal(err)
	}
	name := "questrace-missing-" + hex.EncodeToString(unknown) + ".local"
	if a := ask(t, conn, name, 1500*time.Millisecond); a != nil {
		t.Fatalf("%s should not be answered, got %s", name, a.A)
	}
}

// TestHostnameRespondersSharePort runs two responders at once, which is what
// every extra network interface does on the real machine: each one needs its own
// socket on port 5353 that receives the group traffic and still answers. This is
// the property the single shared socket had to fake with control messages, so it
// is worth pinning down on a machine where the second interface cannot be
// created.
func TestHostnameRespondersSharePort(t *testing.T) {
	if os.Getenv("QUESTRACE_MDNS_LIVE") != "1" {
		t.Skip("set QUESTRACE_MDNS_LIVE=1 to run the live mDNS tests")
	}
	ifaces, _ := Interfaces("")
	if len(ifaces) == 0 {
		t.Fatal("no interface with a private IPv4 address")
	}
	suffix := make([]byte, 4)
	if _, err := rand.Read(suffix); err != nil {
		t.Fatal(err)
	}
	names := []string{
		"questrace-test-first-" + hex.EncodeToString(suffix) + ".local",
		"questrace-test-second-" + hex.EncodeToString(suffix) + ".local",
	}
	for _, name := range names {
		responder, err := startHostnameResponder(name, ifaces)
		if err != nil {
			t.Fatalf("startHostnameResponder(%s): %v", name, err)
		}
		defer responder.Close()
	}
	conn := liveQueryConn(t, ifaces[0])
	for _, name := range names {
		a := ask(t, conn, name, 5*time.Second)
		if a == nil {
			t.Fatalf("no answer for %s", name)
		}
		if !a.A.IsPrivate() {
			t.Fatalf("%s: answered a non-private address %s", name, a.A)
		}
		t.Logf("answered %s -> %s", name, a.A)
	}
}
