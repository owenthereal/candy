package dns

import (
	"context"
	"fmt"
	"net"
	"time"

	"github.com/miekg/dns"
	"github.com/oklog/run"
	"github.com/owenthereal/candy"
	"go.uber.org/zap"
)

type Config struct {
	Addr    string
	TLDs    []string
	LocalIP bool
	Logger  *zap.Logger
}

func New(cfg Config) candy.DNSServer {
	return &dnsServer{
		cfg: cfg,
	}
}

type dnsServer struct {
	cfg Config
}

func (d *dnsServer) Run(ctx context.Context) error {
	d.cfg.Logger.Info("starting DNS server", zap.Any("cfg", d.cfg))
	defer d.cfg.Logger.Info("shutting down DNS server")

	mux := dns.NewServeMux()
	for _, tld := range d.cfg.TLDs {
		mux.HandleFunc(tld+".", d.handleDNS)
	}

	ctx, cancel := context.WithCancel(ctx)
	var g run.Group
	{
		udp := &dns.Server{
			Handler: mux,
			Addr:    d.cfg.Addr,
			Net:     "udp",
		}
		g.Add(func() error {
			return udp.ListenAndServe()
		}, func(err error) {
			_ = udp.ShutdownContext(ctx)
		})
	}
	{
		tcp := &dns.Server{
			Handler: mux,
			Addr:    d.cfg.Addr,
			Net:     "tcp",
		}
		g.Add(func() error {
			return tcp.ListenAndServe()
		}, func(err error) {
			_ = tcp.ShutdownContext(ctx)
		})
	}
	{
		g.Add(func() error {
			<-ctx.Done()
			return ctx.Err()
		}, func(err error) {
			cancel()
		})
	}

	return g.Run()
}

func (d *dnsServer) handleDNS(w dns.ResponseWriter, r *dns.Msg) {
	dom := r.Question[0].Name

	m := new(dns.Msg)
	m.SetReply(r)

	var (
		a   net.IP
		err error
	)

	if d.cfg.LocalIP {
		a, err = localV4IP()
		if err != nil {
			d.cfg.Logger.Error("error getting local v4 IP", zap.Error(err))
			_ = w.WriteMsg(m)
			return
		}
	} else {
		a = clientIP(w)
	}

	var (
		rr dns.RR
		v4 bool = a.To4() != nil
	)

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

	_ = w.WriteMsg(m)
}

func clientIP(w dns.ResponseWriter) net.IP {
	var a net.IP

	if ip, ok := w.RemoteAddr().(*net.UDPAddr); ok {
		a = ip.IP
	}

	if ip, ok := w.RemoteAddr().(*net.TCPAddr); ok {
		a = ip.IP
	}

	return a
}

func localV4IP() (net.IP, error) {
	ifaces, err := net.Interfaces()
	if err != nil {
		return nil, err
	}

	for _, iface := range ifaces {
		if iface.Flags&net.FlagUp == 0 {
			continue // interface down
		}
		if iface.Flags&net.FlagLoopback != 0 {
			continue // loopback interface
		}
		addrs, err := iface.Addrs()
		if err != nil {
			return nil, err
		}
		for _, addr := range addrs {
			var ip net.IP
			switch v := addr.(type) {
			case *net.IPNet:
				ip = v.IP
			case *net.IPAddr:
				ip = v.IP
			}
			if ip == nil || ip.IsLoopback() {
				continue
			}

			ip = ip.To4()
			if ip == nil {
				continue // not an ipv4 address
			}

			return ip, nil
		}
	}

	return nil, fmt.Errorf("no external IP")
}
