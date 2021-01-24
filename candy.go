package candy

import "context"

type ProxyServerConfig struct {
	Hosts []ProxyHost
}

type ProxyHost struct {
	Host     string
	Upstream string
}

type ProxyServer interface {
	Start(ctx context.Context, cfg ProxyServerConfig) error
}

type DNSServerConfig struct {
	Addr    string
	Domains []string
}

type DNSServer interface {
	Start(ctx context.Context, cfg DNSServerConfig) error
}
