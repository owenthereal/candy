package main

import (
	"os"

	"github.com/owenthereal/candy"
	"github.com/owenthereal/candy/caddy"
	"github.com/owenthereal/candy/dns"
	"github.com/owenthereal/candy/fswatch"
	"go.uber.org/zap"
)

func main() {
	dir := "/Users/owen/.candy"
	tlds := []string{"meroxa"}

	if err := os.MkdirAll(dir, 0o0644); err != nil {
		candy.Log().Fatal("failed to start Candy server", zap.Error(err))
	}

	svr := candy.Server{
		Proxy: caddy.New(caddy.Config{
			HTTPAddr:  ":80",
			HTTPSAddr: ":443",
			AdminAddr: "localhost:22019",
			TLDs:      tlds,
			DomainDir: dir,
		}),
		DNS: dns.New(dns.Config{
			Addr: ":25353",
			TLDs: tlds,
		}),
		Watcher: fswatch.New(fswatch.Config{DomainDir: dir}),
	}

	if err := svr.Start(); err != nil {
		candy.Log().Fatal("failed to start Candy server", zap.Error(err))
	}
}
