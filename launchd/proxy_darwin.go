package launchd

import (
	"context"
	"net"

	"inet.af/tcpproxy"
)

type SocketProxyConfig struct {
	LaunchdSocketName string
	UnixSocketPath    string
}

func NewSocketProxy(cfg SocketProxyConfig) ([]tcpproxy.Proxy, error) {
	lns, err := socketListeners(cfg.LaunchdSocketName)
	if err != nil {
		return nil, err
	}

	var result []tcpproxy.Proxy
	for _, ln := range lns {
		proxy := tcpproxy.Proxy{
			ListenFunc: func(net, laddr string) (net.Listener, error) {
				return ln, nil
			},
		}
		dproxy := &tcpproxy.DialProxy{
			DialContext: func(ctx context.Context, network, address string) (net.Conn, error) {
				return net.Dial("unix", cfg.UnixSocketPath)
			},
		}
		proxy.AddRoute("", dproxy)

		result = append(result, proxy)
	}

	return result, nil
}
