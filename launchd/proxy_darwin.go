package launchd

import (
	"context"
	"net"

	"github.com/bored-engineer/go-launchd"
	"github.com/owenthereal/candy"
	"go.uber.org/zap"
	"inet.af/tcpproxy"
)

type SocketProxyConfig struct {
	LaunchdSocketName string
	UnixSocketPath    string
}

func NewSocketProxy(cfg SocketProxyConfig) (*tcpproxy.Proxy, error) {
	logger := candy.Log().Named("socket-proxy")

	ln, err := launchd.Activate(cfg.LaunchdSocketName)
	if err != nil {
		return nil, err
	}

	proxy := &tcpproxy.Proxy{
		ListenFunc: func(net, laddr string) (net.Listener, error) {
			return ln, nil
		},
	}
	dproxy := &tcpproxy.DialProxy{
		DialContext: func(ctx context.Context, network, address string) (net.Conn, error) {
			return net.Dial("unix", cfg.UnixSocketPath)
		},
		OnDialError: func(src net.Conn, err error) {
			logger.Error("dial error", zap.Error(err))
		},
	}
	proxy.AddRoute("", dproxy)

	return proxy, nil
}
