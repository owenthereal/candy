package candy

import (
	"context"

	"github.com/oklog/run"
)

type Server struct {
	Proxy ProxyServer
	DNS   DNSServer
}

func (s *Server) Start(ctx context.Context) error {
	var g run.Group
	{
		cfg := ProxyServerConfig{
			Hosts: []ProxyHost{
				{
					Host:     "api.meroxa",
					Upstream: "192.168.64.36:30784",
				},
				{
					Host:     "logmgr.meroxa",
					Upstream: "192.168.64.36:31525",
				},
			},
		}

		g.Add(func() error {
			return s.Proxy.Start(ctx, cfg)
		}, func(error) {
		})
	}
	{
		cfg := DNSServerConfig{
			Domains: []string{"meroxa"},
		}

		g.Add(func() error {
			return s.DNS.Start(ctx, cfg)
		}, func(error) {
		})
	}

	return g.Run()
}
