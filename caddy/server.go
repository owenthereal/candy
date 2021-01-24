package caddy

import (
	"context"
	"encoding/json"

	"github.com/caddyserver/caddy/v2"
	"github.com/caddyserver/caddy/v2/caddyconfig"
	"github.com/caddyserver/caddy/v2/modules/caddyhttp"
	"github.com/caddyserver/caddy/v2/modules/caddyhttp/reverseproxy"
	"github.com/owenthereal/candy"
)

type Config struct {
	Addr string
}

func New(cfg Config) candy.ProxyServer {
	return &caddyServer{Config: cfg}
}

type caddyServer struct {
	Config Config
}

func (c *caddyServer) Start(ctx context.Context, cfg candy.ProxyServerConfig) error {
	// TODO: Parse host.Upstream: 8080, 127.0.0.1:8080, JSON

	var routes caddyhttp.RouteList
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
		}

		routes = append(routes, route)
	}

	server := &caddyhttp.Server{
		AutoHTTPS: &caddyhttp.AutoHTTPSConfig{Disabled: false},
		Routes:    routes,
		Listen:    []string{c.Config.Addr},
	}

	httpApp := caddyhttp.App{
		Servers: map[string]*caddyhttp.Server{"proxy": server},
	}

	ccfg := &caddy.Config{
		Admin: &caddy.AdminConfig{Listen: ":22019"},
		AppsRaw: caddy.ModuleMap{
			"http": caddyconfig.JSON(httpApp, nil),
		},
	}

	return caddy.Run(ccfg)
}
