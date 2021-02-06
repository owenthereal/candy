package caddy

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/caddyserver/caddy/v2"
	"github.com/caddyserver/caddy/v2/caddyconfig"
	"github.com/caddyserver/caddy/v2/modules/caddyhttp"
	"github.com/caddyserver/caddy/v2/modules/caddyhttp/reverseproxy"
	"github.com/caddyserver/caddy/v2/modules/caddytls"
	"github.com/owenthereal/candy"
	"go.uber.org/zap"
)

var (
	caddyAPITimeout = 5 * time.Second
)

type Config struct {
	HTTPAddr  string
	HTTPSAddr string
	AdminAddr string
	TLDs      []string
	HostRoot  string
	Logger    *zap.Logger
}

func New(cfg Config) candy.ProxyServer {
	return &caddyServer{
		cfg: cfg,
		apps: candy.NewAppService(candy.AppServiceConfig{
			TLDs:     cfg.TLDs,
			HostRoot: cfg.HostRoot,
		}),
	}
}

type caddyServer struct {
	ctx  context.Context
	cfg  Config
	apps *candy.AppService

	caddyCfg      *caddy.Config
	caddyCfgMutex sync.Mutex
}

func (c *caddyServer) Run(ctx context.Context) error {
	c.ctx = ctx

	if err := c.startServer(); err != nil {
		return err
	}

	<-ctx.Done()

	if err := c.stopServer(); err != nil {
		return err
	}

	return ctx.Err()
}

func (c *caddyServer) startServer() error {
	c.cfg.Logger.Info("starting Caddy server", zap.Any("cfg", c.cfg))

	c.caddyCfgMutex.Lock()
	defer c.caddyCfgMutex.Unlock()

	caddy.TrapSignals()

	ccfg, err := c.loadConfig()
	if err != nil {
		return fmt.Errorf("error loading Caddy config: %w", err)
	}

	c.caddyCfg = ccfg

	return caddy.Run(ccfg)
}

func (c *caddyServer) stopServer() error {
	c.cfg.Logger.Info("shutting down Caddy server")

	return c.apiRequest(context.Background(), http.MethodPost, "/stop", nil)
}

func (c *caddyServer) Reload() error {
	c.cfg.Logger.Info("reloading Caddy server")

	c.caddyCfgMutex.Lock()
	defer c.caddyCfgMutex.Unlock()

	ccfg, err := c.loadConfig()
	if err != nil {
		return fmt.Errorf("error reloading Caddy config: %w", err)
	}

	if jsonEqual(c.caddyCfg, ccfg) {
		c.cfg.Logger.Info("Caddy server unchanged")
		return nil
	}

	if err := c.apiRequest(c.ctx, http.MethodPost, "/load", ccfg); err != nil {
		return err
	}

	c.caddyCfg = ccfg

	return nil
}

func (c *caddyServer) loadConfig() (*caddy.Config, error) {
	apps, err := c.apps.FindApps()
	if err != nil {
		return nil, fmt.Errorf("error loading apps: %w", err)
	}

	return c.buildConfig(apps), nil
}

func (c *caddyServer) buildConfig(apps []candy.App) *caddy.Config {
	httpServer := &caddyhttp.Server{
		Routes: caddyRoutes(
			reverseproxy.HTTPTransport{
				Versions: []string{"1.1", "2", "h2c"},
			},
			apps,
		),
		Listen:    []string{c.cfg.HTTPAddr},
		AutoHTTPS: &caddyhttp.AutoHTTPSConfig{Disabled: true},
		AllowH2C:  true,
	}

	httpsServer := &caddyhttp.Server{
		Routes: caddyRoutes(
			reverseproxy.HTTPTransport{
				Versions: []string{"1.1", "2"},
			},
			apps,
		),
		Listen: []string{c.cfg.HTTPSAddr},
	}

	// Best efforts of parsing corresponding port from addr
	// If they are 0, Caddy will use the default ports
	// See https://caddyserver.com/docs/json/apps/http/http_port
	_, httpPortStr, _ := net.SplitHostPort(c.cfg.HTTPAddr)
	_, httpsPortStr, _ := net.SplitHostPort(c.cfg.HTTPSAddr)
	httpPort, _ := strconv.Atoi(httpPortStr)
	httpsPort, _ := strconv.Atoi(httpsPortStr)

	httpApp := caddyhttp.App{
		HTTPPort:  httpPort,
		HTTPSPort: httpsPort,
		Servers: map[string]*caddyhttp.Server{
			"http":  httpServer,
			"https": httpsServer,
		},
	}

	tls := caddytls.TLS{
		Automation: &caddytls.AutomationConfig{
			Policies: []*caddytls.AutomationPolicy{
				{
					Subjects: appHosts(apps),
					IssuersRaw: []json.RawMessage{
						caddyconfig.JSONModuleObject(caddytls.InternalIssuer{}, "module", "internal", nil),
					},
				},
			},
		},
	}

	return &caddy.Config{
		Admin: &caddy.AdminConfig{Listen: c.cfg.AdminAddr},
		AppsRaw: caddy.ModuleMap{
			"http": caddyconfig.JSON(httpApp, nil),
			"tls":  caddyconfig.JSON(tls, nil),
		},
	}
}

func (c *caddyServer) apiRequest(ctx context.Context, method, uri string, v interface{}) error {
	ctx, cancel := context.WithTimeout(ctx, caddyAPITimeout)
	defer cancel()

	parsedAddr, err := caddy.ParseNetworkAddress(c.cfg.AdminAddr)
	if err != nil || parsedAddr.PortRangeSize() > 1 {
		return fmt.Errorf("invalid admin address %s: %v", c.cfg.AdminAddr, err)
	}
	origin := parsedAddr.JoinHostPort(0)
	if parsedAddr.IsUnixNetwork() {
		origin = "unixsocket" // hack so that http.NewRequest() is happy
	}

	var body io.Reader
	if v != nil {
		b, err := json.Marshal(v)
		if err != nil {
			return fmt.Errorf("error marshaling JSON: %w", err)
		}

		body = bytes.NewReader(b)
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

func appHosts(apps []candy.App) []string {
	var hosts []string

	for _, app := range apps {
		hosts = append(hosts, app.Host)
	}

	return hosts
}

func caddyRoutes(tr reverseproxy.HTTPTransport, apps []candy.App) []caddyhttp.Route {
	var routes caddyhttp.RouteList

	for _, app := range apps {
		handler := reverseproxy.Handler{
			TransportRaw: caddyconfig.JSONModuleObject(tr, "protocol", "http", nil),
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
	}

	return routes
}

func jsonEqual(v1, v2 interface{}) bool {
	b1, err := json.Marshal(v1)
	if err != nil {
		return false
	}

	b2, err := json.Marshal(v2)
	if err != nil {
		return false
	}

	return bytes.Equal(b1, b2)
}
