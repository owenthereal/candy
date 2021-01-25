package candy

import (
	"github.com/caddyserver/caddy/v2"
	"go.uber.org/zap"
)

type ProxyServer interface {
	Start() error
	Reload() error
	Shutdown() error
}

type DNSServer interface {
	Start() error
	Shutdown() error
}

type WatcherHandleFunc func()

type Watcher interface {
	Watch(WatcherHandleFunc) error
	Shutdown() error
}

func Log() *zap.Logger {
	return caddy.Log().Named("candy")
}
