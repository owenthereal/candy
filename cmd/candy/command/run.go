package command

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/owenthereal/candy"
	"github.com/owenthereal/candy/caddy"
	"github.com/owenthereal/candy/dns"
	"github.com/owenthereal/candy/fswatch"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

func Run() *cobra.Command {
	runCmd := &cobra.Command{
		Use:   "run",
		Short: "Starts the Candy process and blocks indefinitely",
		RunE:  runRunE,
	}

	addServerFlags(runCmd)

	return runCmd
}

func addServerFlags(cmd *cobra.Command) {
	cmd.Flags().String("host-root", filepath.Join(homeDir, ".candy"), "Path to the directory containing applications that will be served by Candy")
	cmd.Flags().StringSlice("domain", []string{"test"}, "The top-level domains for which Candy will respond to DNS queries")
	cmd.Flags().String("http-addr", ":80", "The Proxy server HTTP address")
	cmd.Flags().String("https-addr", ":443", "The Proxy server HTTPS address")
	cmd.Flags().String("admin-addr", "127.0.0.1:22019", "The Proxy server administrative address")
	cmd.Flags().String("dns-addr", ":25353", "The DNS server address")
	cmd.Flags().Bool("dns-local-ip", false, "DNS server responds DNS queries with local IP instead of 127.0.0.1")
}

func runRunE(c *cobra.Command, args []string) error {
	return startServer(c)
}

func startServer(c *cobra.Command) error {
	var cfg candy.ServerConfig
	if err := candy.LoadConfig(
		flagRootCfgFile,
		c,
		[]string{
			"host-root",
			"domain",
			"http-addr",
			"https-addr",
			"admin-addr",
			"dns-addr",
		},
		&cfg,
	); err != nil {
		return err
	}

	candy.Log().Info("using config", zap.Reflect("config", cfg))

	if err := os.MkdirAll(cfg.HostRoot, 0o0755); err != nil {
		return fmt.Errorf("failed to create host directory: %w", err)
	}

	svr := candy.Server{
		Proxy: caddy.New(caddy.Config{
			HTTPAddr:  cfg.HttpAddr,
			HTTPSAddr: cfg.HttpsAddr,
			AdminAddr: cfg.AdminAddr,
			TLDs:      cfg.Domain,
			HostRoot:  cfg.HostRoot,
			Logger:    candy.Log().Named("caddy"),
		}),
		DNS: dns.New(dns.Config{
			Addr:    cfg.DnsAddr,
			TLDs:    cfg.Domain,
			LocalIP: cfg.DnsLocalIp,
			Logger:  candy.Log().Named("dns"),
		}),
		Watcher: fswatch.New(fswatch.Config{
			HostRoot: cfg.HostRoot,
			Logger:   candy.Log().Named("fswatch"),
		}),
	}

	return svr.Start()
}
