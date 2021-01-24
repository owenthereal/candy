package dns

import (
	"context"
	"net"
	"time"

	"github.com/miekg/dns"
	"github.com/oklog/run"
	"github.com/owenthereal/candy"
)

type Config struct {
	Addr string
}

func New(cfg Config) candy.DNSServer {
	return &dnsServer{Config: cfg}
}

type dnsServer struct {
	Config Config
}

func (d *dnsServer) Start(ctx context.Context, cfg candy.DNSServerConfig) error {
	for _, domain := range cfg.Domains {
		dns.HandleFunc(domain+".", d.handleDNS)
	}

	var g run.Group
	{
		udp := &dns.Server{Addr: d.Config.Addr, Net: "udp"}
		g.Add(func() error {
			return udp.ListenAndServe()
		}, func(error) {
			udp.ShutdownContext(ctx)
		})
	}
	{
		tcp := &dns.Server{Addr: d.Config.Addr, Net: "tcp"}
		g.Add(func() error {
			return tcp.ListenAndServe()
		}, func(error) {
			tcp.ShutdownContext(ctx)
		})
	}

	return g.Run()
}

func (d *dnsServer) handleDNS(w dns.ResponseWriter, r *dns.Msg) {
	var (
		v4 bool
		rr dns.RR
		a  net.IP
	)

	dom := r.Question[0].Name

	m := new(dns.Msg)
	m.SetReply(r)

	if ip, ok := w.RemoteAddr().(*net.UDPAddr); ok {
		a = ip.IP
		v4 = a.To4() != nil
	}
	if ip, ok := w.RemoteAddr().(*net.TCPAddr); ok {
		a = ip.IP
		v4 = a.To4() != nil
	}

	if v4 {
		rr = &dns.A{
			Hdr: dns.RR_Header{Name: dom, Rrtype: dns.TypeA, Class: dns.ClassINET, Ttl: 0},
			A:   a.To4(),
		}
	} else {
		rr = &dns.AAAA{
			Hdr:  dns.RR_Header{Name: dom, Rrtype: dns.TypeAAAA, Class: dns.ClassINET, Ttl: 0},
			AAAA: a,
		}
	}

	switch r.Question[0].Qtype {
	case dns.TypeAAAA, dns.TypeA:
		m.Answer = append(m.Answer, rr)
	}

	if r.IsTsig() != nil {
		if w.TsigStatus() == nil {
			m.SetTsig(r.Extra[len(r.Extra)-1].(*dns.TSIG).Hdr.Name, dns.HmacMD5, 300, time.Now().Unix())
		}
	}

	w.WriteMsg(m)
}
