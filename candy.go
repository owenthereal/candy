package candy

import (
	"github.com/caddyserver/caddy/v2"
	"github.com/owenthereal/candy/runnable"
	"go.uber.org/zap"
)

type ProxyServer interface {
	runnable.Runable
	Reload() error
}

type DNSServer interface {
	runnable.Runable
}

type Watcher interface {
	runnable.Runable
}

func Log() *zap.Logger {
	return caddy.Log().Named("candy")
}
