// +build darwin

package cmd

import (
	"errors"
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"path/filepath"
	"strings"

	"github.com/owenthereal/candy"
	"github.com/owenthereal/candy/server"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

const (
	resolverDir  = "/etc/resolver"
	resolverTmpl = `domain %s
nameserver %s
port %s
search_order 1
timeout 5`
)

var setupCmd = &cobra.Command{
	Use:   "setup",
	Short: "Run system setup for Mac",
	RunE:  setupRunE,
}

func init() {
	rootCmd.AddCommand(setupCmd)
	setupCmd.Flags().StringSlice("domain", defaultDomains, "The top-level domains for which Candy will respond to DNS queries")
	setupCmd.Flags().String("dns-addr", defaultDNSAddr, "The DNS server address")
}

func setupRunE(c *cobra.Command, args []string) error {
	err := runSetupRunE(c, args)
	if err != nil {
		if errors.Is(err, os.ErrPermission) {
			candy.Log().Error(fmt.Sprintf("requiring superuser privileges, rerun with `sudo %s`", strings.Join(os.Args, " ")))
		}
	}

	return err
}

func runSetupRunE(c *cobra.Command, args []string) error {
	var cfg server.Config
	if err := candy.LoadConfig(
		flagRootCfgFile,
		c,
		[]string{
			"domain",
			"dns-addr",
		},
		&cfg,
	); err != nil {
		return err
	}

	host, port, err := net.SplitHostPort(cfg.DnsAddr)
	if err != nil {
		return err
	}

	if err := os.MkdirAll(resolverDir, 0o755); err != nil {
		return err
	}

	var logger = candy.Log()

	for _, domain := range cfg.Domain {
		file := filepath.Join(resolverDir, "candy-"+domain)
		content := fmt.Sprintf(resolverTmpl, domain, host, port)

		b, err := ioutil.ReadFile(file)
		if err == nil {
			if string(b) == content {
				logger.Info("resolver configuration file unchanged", zap.String("file", file))
				continue
			}
		}

		logger.Info("writing resolver configuration file", zap.String("file", file))
		if err := ioutil.WriteFile(file, []byte(content), 0o644); err != nil {
			return err
		}
	}

	return nil
}
