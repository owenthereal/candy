package candy

import (
	"github.com/oklog/run"
	"go.uber.org/zap"
)

type ServerConfig struct {
	HostRoot   string   `mapstructure:"host-root"`
	Domain     []string `mapstructure:"domain"`
	HttpAddr   string   `mapstructure:"http-addr"`
	HttpsAddr  string   `mapstructure:"https-addr"`
	AdminAddr  string   `mapstructure:"admin-addr"`
	DnsUdpAddr string   `mapstructure:"dns-udp-addr"`
	DnsTcpAddr string   `mapstructure:"dns-tcp-addr"`
	DnsLocalIp bool     `mapstructure:"dns-local-ip"`
}

type Server struct {
	Proxy   ProxyServer
	DNS     DNSServer
	Watcher Watcher
}

func (s *Server) Start() error {
	var g run.Group
	{
		g.Add(func() error {
			return s.Proxy.Start()
		}, func(err error) {
			_ = s.Proxy.Shutdown()
		})
	}
	{
		g.Add(func() error {
			return s.DNS.Start()
		}, func(err error) {
			_ = s.DNS.Shutdown()
		})
	}
	{
		g.Add(func() error {
			return s.Watcher.Watch(func() {
				if err := s.Proxy.Reload(); err != nil {
					Log().Error("error reloading proxy", zap.Error(err))
				}
			})
		}, func(err error) {
			_ = s.Watcher.Shutdown()
		})
	}

	return g.Run()
}
