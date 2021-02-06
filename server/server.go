package server

import (
	"context"
	"fmt"

	"github.com/owenthereal/candy"
	"github.com/owenthereal/candy/caddy"
	"github.com/owenthereal/candy/dns"
	"github.com/owenthereal/candy/runnable"
	"github.com/owenthereal/candy/watch"
	"go.uber.org/zap"
)

type Config struct {
	HostRoot   string   `mapstructure:"host-root"`
	Domain     []string `mapstructure:"domain"`
	HttpAddr   string   `mapstructure:"http-addr"`
	HttpsAddr  string   `mapstructure:"https-addr"`
	AdminAddr  string   `mapstructure:"admin-addr"`
	DnsAddr    string   `mapstructure:"dns-addr"`
	DnsLocalIp bool     `mapstructure:"dns-local-ip"`
}

func (c Config) Validate() error {
	if c.HostRoot == "" {
		return fmt.Errorf("--host-root is required")
	}

	if len(c.Domain) == 0 {
		return fmt.Errorf("--domain is required")
	}

	if c.HttpAddr == "" {
		return fmt.Errorf("--http-addr is required")
	}

	if c.HttpsAddr == "" {
		return fmt.Errorf("--https-addr is required")
	}

	if c.AdminAddr == "" {
		return fmt.Errorf("--admin-addr is required")
	}

	if c.DnsAddr == "" {
		return fmt.Errorf("--dns-addr is required")
	}

	return nil
}

func New(cfg Config) *Server {
	return &Server{cfg: cfg}
}

type Server struct {
	cfg Config
}

func (s *Server) Run(ctx context.Context) error {
	logger := candy.Log().Named("server")

	caddySvr := caddy.New(caddy.Config{
		HTTPAddr:  s.cfg.HttpAddr,
		HTTPSAddr: s.cfg.HttpsAddr,
		AdminAddr: s.cfg.AdminAddr,
		TLDs:      s.cfg.Domain,
		HostRoot:  s.cfg.HostRoot,
		Logger:    logger.Named("caddy"),
	})

	dns := dns.New(dns.Config{
		Addr:    s.cfg.DnsAddr,
		TLDs:    s.cfg.Domain,
		LocalIP: s.cfg.DnsLocalIp,
		Logger:  logger.Named("dns"),
	})

	watchLogger := logger.Named("watch")
	watcher := watch.New(watch.Config{
		HostRoot: s.cfg.HostRoot,
		HandleFunc: func() {
			if err := caddySvr.Reload(); err != nil {
				watchLogger.Error("error reloading Caddy server", zap.Error(err))
			}
		},
		Logger: watchLogger,
	})

	return runnable.RunWithContext(ctx, []runnable.Runable{caddySvr, dns, watcher})
}
