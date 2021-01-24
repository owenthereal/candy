package caddy

import (
	"context"
	"encoding/json"

	"github.com/caddyserver/caddy/v2"
	"github.com/caddyserver/caddy/v2/caddyconfig"
	"github.com/caddyserver/caddy/v2/modules/caddyhttp"
	"github.com/caddyserver/caddy/v2/modules/caddyhttp/reverseproxy"
	"github.com/caddyserver/caddy/v2/modules/caddytls"
	"github.com/owenthereal/candy"
)

type Config struct {
	HTTPAddr  string
	HTTPSAddr string
	AdminAddr string
}

func New(cfg Config) candy.ProxyServer {
	return &caddyServer{Config: cfg}
}

type caddyServer struct {
	Config Config
}

func (c *caddyServer) Start(ctx context.Context, cfg candy.ProxyServerConfig) error {
	// TODO: Parse host.Upstream: 8080, 127.0.0.1:8080, JSON

	var (
		routes caddyhttp.RouteList
		hosts  []string
	)
	for _, host := range cfg.Hosts {
		ht := reverseproxy.HTTPTransport{}
		handler := reverseproxy.Handler{
			TransportRaw: caddyconfig.JSONModuleObject(ht, "protocol", "http", nil),
			Upstreams:    reverseproxy.UpstreamPool{{Dial: host.Upstream}},
		}
		route := caddyhttp.Route{
			HandlersRaw: []json.RawMessage{
				caddyconfig.JSONModuleObject(handler, "handler", "reverse_proxy", nil),
			},
			MatcherSetsRaw: []caddy.ModuleMap{
				{
					"host": caddyconfig.JSON(caddyhttp.MatchHost{host.Host}, nil),
				},
			},
			Terminal: true,
		}

		routes = append(routes, route)
		hosts = append(hosts, host.Host)
	}

	server := &caddyhttp.Server{
		Routes: routes,
		Listen: []string{c.Config.HTTPAddr, c.Config.HTTPSAddr},
	}

	httpApp := caddyhttp.App{
		Servers: map[string]*caddyhttp.Server{"candy": server},
	}

	tls := caddytls.TLS{
		Automation: &caddytls.AutomationConfig{
			Policies: []*caddytls.AutomationPolicy{
				{
					Subjects: hosts,
					IssuersRaw: []json.RawMessage{
						caddyconfig.JSONModuleObject(caddytls.InternalIssuer{}, "module", "internal", nil),
					},
				},
			},
		},
	}

	ccfg := &caddy.Config{
		Admin: &caddy.AdminConfig{Listen: c.Config.AdminAddr},
		AppsRaw: caddy.ModuleMap{
			"http": caddyconfig.JSON(httpApp, nil),
			"tls":  caddyconfig.JSON(tls, nil),
		},
	}

	return caddy.Run(ccfg)
}
