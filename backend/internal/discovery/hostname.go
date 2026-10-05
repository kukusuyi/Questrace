package discovery

import (
	"errors"
	"log"
	"net"
	"strings"
	"sync"

	"github.com/miekg/dns"
	"golang.org/x/net/ipv4"
)

// zeroconf publishes DNS-SD records but does not answer standalone host queries.
// This companion responder makes the advertised .local URL directly resolvable.
//
// It opens one socket per interface instead of one shared socket driven by
// per-packet control messages. golang.org/x/net does not implement ControlMessage
// on Windows: ipv4/control_windows.go returns errNotImplemented from
// SetControlMessage and ipv4/payload_nocmsg.go leaves the received control
// message nil, so code that reads cm.IfIndex or sends a ControlMessage{IfIndex}
// can never answer anything there. With a socket per interface the ingress
// interface is implied by the socket itself, and replies leave through that
// socket's IP_MULTICAST_IF (IP_MULTICAST_IF / IP_ADD_MEMBERSHIP are implemented
// for Windows in ipv4/sys_windows.go + ipv4/sys_asmreq.go), which behaves the
// same on Windows, macOS and Linux.
type hostnameResponder struct {
	hostname string
	conns    []*ifaceResponder
	wg       sync.WaitGroup
	once     sync.Once
}

// ifaceResponder is one socket bound to the mDNS port and joined, on a single
// interface, to the mDNS group.
type ifaceResponder struct {
	nic  net.Interface
	ips  []net.IP
	conn *ipv4.PacketConn
}

// mdnsGroup is the IPv4 multicast group and port defined for mDNS (RFC 6762).
var mdnsGroup = &net.UDPAddr{IP: net.IPv4(224, 0, 0, 251), Port: 5353}

// hostnameAnswer builds the A-record response for a standalone host query, or
// nil when the query does not ask for this hostname (or is not a query at all).
func hostnameAnswer(query *dns.Msg, hostname string, ips []net.IP) *dns.Msg {
	if query.Response {
		return nil
	}
	matched := false
	for _, q := range query.Question {
		if strings.EqualFold(q.Name, dns.Fqdn(hostname)) && (q.Qclass&0x7fff) == dns.ClassINET && (q.Qtype == dns.TypeA || q.Qtype == dns.TypeANY) {
			matched = true
		}
	}
	if !matched {
		return nil
	}
	reply := new(dns.Msg)
	reply.Response = true
	reply.Authoritative = true
	for _, ip := range ips {
		if ip.To4() != nil {
			reply.Answer = append(reply.Answer, &dns.A{Hdr: dns.RR_Header{Name: dns.Fqdn(hostname), Rrtype: dns.TypeA, Class: dns.ClassINET | 0x8000, Ttl: 120}, A: ip.To4()})
		}
	}
	return reply
}

// privateIPv4 lists the advertisable IPv4 addresses of one interface, using the
// same filter as Interfaces().
func privateIPv4(nic net.Interface) []net.IP {
	addrs, err := nic.Addrs()
	if err != nil {
		return nil
	}
	var ips []net.IP
	for _, addr := range addrs {
		ip, _, err := net.ParseCIDR(addr.String())
		if err != nil || ip.To4() == nil || !ip.IsPrivate() {
			continue
		}
		ips = append(ips, ip)
	}
	return ips
}

// startHostnameResponder listens for standalone hostname queries on every usable
// interface. It only fails when no interface could be served at all; a single
// unusable interface (for example a virtual adapter) is logged and skipped so it
// cannot disable the responder on the remaining interfaces.
func startHostnameResponder(hostname string, ifaces []net.Interface) (*hostnameResponder, error) {
	responder := &hostnameResponder{hostname: hostname}
	var firstErr error
	note := func(nic net.Interface, err error) {
		log.Printf("discovery: hostname responder: %s: %v", nic.Name, err)
		if firstErr == nil {
			firstErr = err
		}
	}
	for _, nic := range ifaces {
		ips := privateIPv4(nic)
		if len(ips) == 0 {
			continue
		}
		// Binding an mDNS multicast address makes net set SO_REUSEADDR and bind
		// to the wildcard, so several listeners can share port 5353 (this is the
		// same reason the shared zeroconf socket coexists with this one).
		udp, err := net.ListenUDP("udp4", &net.UDPAddr{IP: net.IPv4(224, 0, 0, 0), Port: 5353})
		if err != nil {
			note(nic, err)
			continue
		}
		conn := ipv4.NewPacketConn(udp)
		if err := conn.SetMulticastTTL(255); err != nil {
			conn.Close()
			note(nic, err)
			continue
		}
		if err := conn.SetMulticastInterface(&nic); err != nil {
			conn.Close()
			note(nic, err)
			continue
		}
		if err := conn.JoinGroup(&nic, mdnsGroup); err != nil {
			conn.Close()
			note(nic, err)
			continue
		}
		ir := &ifaceResponder{nic: nic, ips: ips, conn: conn}
		responder.conns = append(responder.conns, ir)
		responder.wg.Add(1)
		go responder.serve(ir)
	}
	if len(responder.conns) == 0 {
		if firstErr != nil {
			return nil, firstErr
		}
		return nil, errors.New("no interface with a private IPv4 address")
	}
	return responder, nil
}

// serve answers queries received on one interface socket. No control message is
// read or sent: the interface is the one this socket was joined on.
func (r *hostnameResponder) serve(ir *ifaceResponder) {
	defer r.wg.Done()
	buffer := make([]byte, 9000)
	for {
		n, _, source, err := ir.conn.ReadFrom(buffer)
		if err != nil {
			return
		}
		query := new(dns.Msg)
		if query.Unpack(buffer[:n]) != nil {
			continue
		}
		reply := hostnameAnswer(query, r.hostname, ir.ips)
		if reply == nil || len(reply.Answer) == 0 {
			continue
		}
		destination := net.Addr(mdnsGroup)
		if remote, ok := source.(*net.UDPAddr); ok && remote.Port != 5353 {
			// RFC 6762 section 6.7: legacy unicast queries get a unicast answer
			// with the query ID echoed and a short TTL.
			destination = source
			reply.Id = query.Id
			reply.Question = query.Question
			for _, record := range reply.Answer {
				record.Header().Class = dns.ClassINET
				record.Header().Ttl = 10
			}
		}
		packet, err := reply.Pack()
		if err != nil {
			continue
		}
		// A nil control message is deliberate: the outgoing interface comes from
		// this socket's IP_MULTICAST_IF. Control messages are unimplemented on
		// Windows and would be ignored there anyway.
		_, _ = ir.conn.WriteTo(packet, nil, destination)
	}
}

// Close announces a goodbye (TTL 0) on every interface and stops the responder.
func (r *hostnameResponder) Close() {
	r.once.Do(func() {
		query := new(dns.Msg)
		query.SetQuestion(dns.Fqdn(r.hostname), dns.TypeA)
		for _, ir := range r.conns {
			reply := hostnameAnswer(query, r.hostname, ir.ips)
			if reply != nil && len(reply.Answer) > 0 {
				for _, record := range reply.Answer {
					record.Header().Ttl = 0
				}
				if packet, err := reply.Pack(); err == nil {
					_, _ = ir.conn.WriteTo(packet, nil, mdnsGroup)
				}
			}
			ir.conn.Close()
		}
	})
	r.wg.Wait()
}
