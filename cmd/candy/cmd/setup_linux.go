// +build linux

package cmd

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/owenthereal/candy"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

const (
	resolvedDir  = "/usr/lib/systemd/resolved.conf.d"
	resolvedFile = "01-candy.conf"
	resolvedTmpl = `[Resolve]
DNS=%s
Domains=%s`
)

var setupCmd = &cobra.Command{
	Use:   "setup",
	Short: "Run system setup for Linux",
	RunE:  setupRunE,
}

func init() {
	rootCmd.AddCommand(setupCmd)
	addDefaultFlags(setupCmd)

	// Hide flags that are not used by setup
	setupCmd.Flags().MarkHidden("http-addr")
	setupCmd.Flags().MarkHidden("https-addr")
	setupCmd.Flags().MarkHidden("admin-addr")
	setupCmd.Flags().MarkHidden("dns-addr")
	setupCmd.Flags().MarkHidden("dns-local-ip")
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
