// +build linux

package command

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/owenthereal/candy"
	"github.com/owenthereal/candy/server"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

const (
	resolvedDir  = "/etc/systemd/resolved.conf.d"
	resolvedFile = "01-candy.conf"
	resolvedTmpl = `[Resolve]
DNS=%s
Domains=%s`
)

func newSetupCmd() *cobra.Command {
	setupCmd := &cobra.Command{
		Use:   "setup",
		Short: "Run system setup for Linux",
		RunE: func(c *cobra.Command, args []string) error {
			err := setupRunE(c, args)
			if err != nil {
				if errors.Is(err, os.ErrPermission) {
					candy.Log().Error(fmt.Sprintf("requiring superuser privileges, rerun with `sudo %s`", strings.Join(os.Args, " ")))
				}
			}

			return err
		},
	}

	setupCmd.Flags().StringSlice("domain", defaultDomains, "The top-level domains for which Candy will respond to DNS queries")
	setupCmd.Flags().String("dns-addr", defaultDNSAddr, "The DNS server address")

	return setupCmd
}

func setupRunE(c *cobra.Command, args []string) error {
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

	if err := os.MkdirAll(resolvedDir, 0o755); err != nil {
		return err
	}

	var (
		file    = filepath.Join(resolvedDir, resolvedFile)
		content = fmt.Sprintf(resolvedTmpl, cfg.DnsAddr, strings.Join(cfg.Domain, " "))
		logger  = candy.Log()
	)

	b, err := ioutil.ReadFile(file)
	if err == nil {
		if string(b) == content {
			logger.Info("network name resolution file unchanged", zap.String("file", file))
			return nil
		}
	}

	logger.Info("writing network name resolution file", zap.String("file", file))
	if err := ioutil.WriteFile(file, []byte(content), 0o644); err != nil {
		return err
	}

	logger.Info("restarting systemd-resolved")
	return execCmd("systemctl", "restart", "systemd-resolved")
}

func execCmd(c ...string) error {
	cmd := exec.Command(c[0], c[1:]...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}
