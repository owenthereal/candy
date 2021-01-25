package candy

import (
	"github.com/oklog/run"
	"go.uber.org/zap"
)

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
			s.Proxy.Shutdown()
		})
	}
	{
		g.Add(func() error {
			return s.DNS.Start()
		}, func(err error) {
			s.DNS.Shutdown()
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
			s.Watcher.Shutdown()
		})
	}

	return g.Run()
}
