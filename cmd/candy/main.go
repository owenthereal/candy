package main

import (
	"github.com/owenthereal/candy"
	"github.com/owenthereal/candy/caddy"
	"github.com/owenthereal/candy/dns"
	"go.uber.org/zap"
)

func main() {
	svr := candy.Server{
		Proxy: caddy.New(caddy.Config{HTTPAddr: ":80", HTTPSAddr: ":443", AdminAddr: "localhost:22019"}),
		DNS:   dns.New(dns.Config{Addr: ":25353"}),
	}

	if err := svr.Start(); err != nil {
		candy.Log().Fatal("failed to start Candy server", zap.Error(err))
	}
}
