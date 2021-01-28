// +build darwin

package command

import (
	"errors"
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"path/filepath"
	"strconv"

	"github.com/owenthereal/candy"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

const (
	resolverTmpl = `domain %s
nameserver %s
port %s
search_order 1
timeout 5`
)

var (
	flagSetupCmdDomains  []string
	flagSetupCmdDNSAddr  string
	flagSetupCmdHostRoot string
)

func Setup() *cobra.Command {
	setupCmd := &cobra.Command{
		Use:   "setup",
		Short: "Run system setup",
		RunE:  setupRunE,
	}

	setupCmd.Flags().StringSliceVar(&flagSetupCmdDomains, "domain", []string{"test"}, "The top-level domains for which Candy will respond to DNS queries")
	setupCmd.Flags().StringVar(&flagSetupCmdDNSAddr, "dns-addr", "127.0.0.1:25353", "The DNS server address")
	setupCmd.Flags().StringVar(&flagSetupCmdHostRoot, "host-root", filepath.Join(homeDir, ".candy"), "Path to the directory containing applications that will be served by Candy")

	return setupCmd
}

func setupRunE(c *cobra.Command, args []string) error {
	host, port, err := net.SplitHostPort(flagSetupCmdDNSAddr)
	if err != nil {
		return err
	}

	logger := candy.Log()

	const rd = "/etc/resolver"
	if err := os.MkdirAll(rd, 0o755); err != nil {
		if errors.Is(err, os.ErrPermission) {
			logger.Error("requiring superuser privileges to create directory", zap.String("dir", rd))
		}

		return err
	}

	var (
		sudo        = os.Getenv("SUDO_USER")
		uid         int
		gid         int
		shouldChown bool
	)

	if sudo != "" {
		var err1, err2 error

		uid, err1 = strconv.Atoi(os.Getenv("SUDO_UID"))
		gid, err2 = strconv.Atoi(os.Getenv("SUDO_GID"))

		shouldChown = err1 == nil && err2 == nil
	}

	if shouldChown {
		if err := os.Chown(rd, uid, gid); err != nil {
			return err
		}
	}

	for _, domain := range flagSetupCmdDomains {
		rf := filepath.Join(rd, "candy-"+domain)
		if _, err := os.Stat(rf); err == nil {
			continue
		}

		logger.Info("Writing DNS resolver config", zap.String("file", rf))

		if err := ioutil.WriteFile(
			rf,
			[]byte(fmt.Sprintf(resolverTmpl, domain, host, port)),
			0o644,
		); err != nil {
			return err
		}

		if shouldChown {
			if err := os.Chown(rf, uid, gid); err != nil {
				return err
			}
		}
	}

	if err := os.MkdirAll(flagSetupCmdHostRoot, 0o0755); err != nil {
		return fmt.Errorf("failed to create host directory: %w", err)
	}

	if shouldChown {
		if err := os.Chown(flagSetupCmdHostRoot, uid, gid); err != nil {
			return err
		}
	}

	return nil
}
