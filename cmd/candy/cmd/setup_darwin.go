//go:build darwin

package cmd

import (
	"errors"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"

	"github.com/owenthereal/candy"
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
	addDefaultFlags(setupCmd)

	// Hide flags that are not used by setup
	_ = setupCmd.Flags().MarkHidden("host-root")
	_ = setupCmd.Flags().MarkHidden("http-addr")
	_ = setupCmd.Flags().MarkHidden("https-addr")
	_ = setupCmd.Flags().MarkHidden("admin-addr")
	_ = setupCmd.Flags().MarkHidden("dns-local-ip")
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
	cfg, err := loadServerConfig(c)
	if err != nil {
		return err
	}

	host, port, err := net.SplitHostPort(cfg.DnsAddr)
	if err != nil {
		return err
	}

	if err := os.MkdirAll(resolverDir, 0o755); err != nil {
		return err
	}

	logger := candy.Log()

	for _, domain := range cfg.Domain {
		file := filepath.Join(resolverDir, "candy-"+domain)
		content := fmt.Sprintf(resolverTmpl, domain, host, port)

		b, err := os.ReadFile(file)
		if err == nil {
			if string(b) == content {
				logger.Info("resolver configuration file unchanged", zap.String("file", file))
				continue
			}
		}

		logger.Info("writing resolver configuration file", zap.String("file", file))
		if err := os.WriteFile(file, []byte(content), 0o644); err != nil {
			return err
		}
	}

	return nil
}
