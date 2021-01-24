package main

import (
	"context"
	"log"

	"github.com/owenthereal/candy"
	"github.com/owenthereal/candy/caddy"
	"github.com/owenthereal/candy/dns"
)

func main() {
	svr := candy.Server{
		Proxy: caddy.New(caddy.Config{HTTPAddr: ":80", HTTPSAddr: ":443", AdminAddr: ":22019"}),
		DNS:   dns.New(dns.Config{Addr: ":25353"}),
	}

	if err := svr.Start(context.Background()); err != nil {
		log.Fatal(err)
	}
}
