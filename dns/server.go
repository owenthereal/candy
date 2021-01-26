package dns

import (
	"context"
	"fmt"
	"net"
	"time"

	"github.com/hashicorp/go-multierror"
	"github.com/miekg/dns"
	"github.com/oklog/run"
	"github.com/owenthereal/candy"
	"go.uber.org/zap"
)

type Config struct {
	Addr    string
	TLDs    []string
	LocalIP bool
}

func New(cfg Config) candy.DNSServer {
	ctx, cancel := context.WithCancel(context.Background())
	return &dnsServer{
		cfg:    cfg,
		udp:    &dns.Server{Addr: cfg.Addr, Net: "udp"},
		tcp:    &dns.Server{Addr: cfg.Addr, Net: "tcp"},
		ctx:    ctx,
		cancel: cancel,
	}
}

type dnsServer struct {
	cfg    Config
	udp    *dns.Server
	tcp    *dns.Server
	ctx    context.Context
	cancel context.CancelFunc
}

func (d *dnsServer) Start() error {
	for _, tld := range d.cfg.TLDs {
		dns.HandleFunc(tld+".", d.handleDNS)
	}

	var g run.Group
	{
		g.Add(func() error {
			return d.udp.ListenAndServe()
		}, func(err error) {
			_ = d.Shutdown()
		})
	}
	{
		g.Add(func() error {
			return d.tcp.ListenAndServe()
		}, func(err error) {
			_ = d.Shutdown()
		})
	}
	{
		g.Add(func() error {
			<-d.ctx.Done()
			return d.ctx.Err()
		}, func(err error) {
			_ = d.Shutdown()
		})
	}

	return g.Run()
}

func (d *dnsServer) Shutdown() error {
	defer d.cancel()

	candy.Log().Info("shutting down DNS server")

	var merr *multierror.Error
	if err := d.udp.Shutdown(); err != nil {
		merr = multierror.Append(merr, err)
	}
	if err := d.tcp.Shutdown(); err != nil {
		merr = multierror.Append(merr, err)
	}

	return merr
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
			candy.Log().Error("error getting local v4 IP", zap.Error(err))
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
