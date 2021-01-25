package candy

import (
	"github.com/caddyserver/caddy/v2"
	"go.uber.org/zap"
)

type ProxyServerConfig struct {
	Hosts []ProxyHost
}

type ProxyHost struct {
	Host     string
	Upstream string
}

type ProxyServer interface {
	Start(cfg ProxyServerConfig) error
	Shutdown() error
}

type DNSServerConfig struct {
	Addr    string
	Domains []string
}

type DNSServer interface {
	Start(cfg DNSServerConfig) error
	Shutdown() error
}

func Log() *zap.Logger {
	return caddy.Log().Named("candy")
}
