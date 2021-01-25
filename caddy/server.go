package caddy

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"time"

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
	TLDs      []string
	DomainDir string
}

func New(cfg Config) candy.ProxyServer {
	ctx, cancel := context.WithCancel(context.Background())

	return &caddyServer{
		cfg: cfg,
		apps: candy.NewAppService(candy.AppServiceConfig{
			TLDs:      cfg.TLDs,
			DomainDir: cfg.DomainDir,
		}),
		ctx:    ctx,
		cancel: cancel,
	}
}

type caddyServer struct {
	cfg  Config
	apps *candy.AppService

	ctx    context.Context
	cancel context.CancelFunc
}

func (c *caddyServer) Start() error {
	apps, err := c.apps.FindApps()
	if err != nil {
		return fmt.Errorf("error loading apps: %w", err)
	}

	caddy.TrapSignals()

	var (
		routes caddyhttp.RouteList
		hosts  []string
	)
	for _, app := range apps {
		ht := reverseproxy.HTTPTransport{}
		handler := reverseproxy.Handler{
			TransportRaw: caddyconfig.JSONModuleObject(ht, "protocol", app.Protocol, nil),
			Upstreams:    reverseproxy.UpstreamPool{{Dial: app.Addr}},
		}
		route := caddyhttp.Route{
			HandlersRaw: []json.RawMessage{
				caddyconfig.JSONModuleObject(handler, "handler", "reverse_proxy", nil),
			},
			MatcherSetsRaw: []caddy.ModuleMap{
				{
					"host": caddyconfig.JSON(caddyhttp.MatchHost{app.Host}, nil),
				},
			},
			Terminal: true,
		}

		routes = append(routes, route)
		hosts = append(hosts, app.Host)
	}

	server := &caddyhttp.Server{
		Routes: routes,
		Listen: []string{c.cfg.HTTPAddr, c.cfg.HTTPSAddr},
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
		Admin: &caddy.AdminConfig{Listen: c.cfg.AdminAddr},
		AppsRaw: caddy.ModuleMap{
			"http": caddyconfig.JSON(httpApp, nil),
			"tls":  caddyconfig.JSON(tls, nil),
		},
	}

	if err := caddy.Run(ccfg); err != nil {
		return err
	}

	select {
	case <-c.ctx.Done():
		return c.ctx.Err()
	}
}

func (c *caddyServer) Reload() error {
	fmt.Println("reload")

	return nil
}

func (c *caddyServer) Shutdown() error {
	candy.Log().Info("shutting down Caddy server")

	ctx, cancel := context.WithTimeout(c.ctx, 5*time.Second)

	defer func() {
		c.cancel()
		cancel()
	}()

	return c.apiRequest(ctx, http.MethodPost, "/stop", nil)
}

func (c *caddyServer) apiRequest(ctx context.Context, method, uri string, body io.Reader) error {
	parsedAddr, err := caddy.ParseNetworkAddress(c.cfg.AdminAddr)
	if err != nil || parsedAddr.PortRangeSize() > 1 {
		return fmt.Errorf("invalid admin address %s: %v", c.cfg.AdminAddr, err)
	}
	origin := parsedAddr.JoinHostPort(0)
	if parsedAddr.IsUnixNetwork() {
		origin = "unixsocket" // hack so that http.NewRequest() is happy
	}

	// form the request
	req, err := http.NewRequestWithContext(ctx, method, "http://"+origin+uri, body)
	if err != nil {
		return fmt.Errorf("making request: %v", err)
	}
	if parsedAddr.IsUnixNetwork() {
		// When listening on a unix socket, the admin endpoint doesn't
		// accept any Host header because there is no host:port for
		// a unix socket's address. The server's host check is fairly
		// strict for security reasons, so we don't allow just any
		// Host header. For unix sockets, the Host header must be
		// empty. Unfortunately, Go makes it impossible to make HTTP
		// requests with an empty Host header... except with this one
		// weird trick. (Hopefully they don't fix it. It's already
		// hard enough to use HTTP over unix sockets.)
		//
		// An equivalent curl command would be something like:
		// $ curl --unix-socket caddy.sock http:/:$REQUEST_URI
		req.URL.Host = " "
		req.Host = ""
	} else {
		req.Header.Set("Origin", origin)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	// make an HTTP client that dials our network type, since admin
	// endpoints aren't always TCP, which is what the default transport
	// expects; reuse is not of particular concern here
	client := http.Client{
		Transport: &http.Transport{
			DialContext: func(_ context.Context, _, _ string) (net.Conn, error) {
				return net.Dial(parsedAddr.Network, parsedAddr.JoinHostPort(0))
			},
		},
		Timeout: 5 * time.Second,
	}

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("performing request: %v", err)
	}
	defer resp.Body.Close()

	// if it didn't work, let the user know
	if resp.StatusCode >= 400 {
		respBody, err := ioutil.ReadAll(io.LimitReader(resp.Body, 1024*10))
		if err != nil {
			return fmt.Errorf("HTTP %d: reading error message: %v", resp.StatusCode, err)
		}
		return fmt.Errorf("caddy responded with error: HTTP %d: %s", resp.StatusCode, respBody)
	}

	return nil
}
