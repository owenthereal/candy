// +build darwin

package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/oklog/run"
	"github.com/owenthereal/candy/launchd"
	"github.com/spf13/cobra"
)

var launchCmd = &cobra.Command{
	Use:   "launch",
	Short: "Starts the Candy process and runs as a launchd daemon (Mac OS)",
	RunE:  launchRunE,
}

func init() {
	rootCmd.AddCommand(launchCmd)
	addServerFlags(launchCmd)
}

func launchRunE(c *cobra.Command, args []string) error {
	httpUnixSocketPath := generateSocketPath("http")
	httpsUnixSocketPath := generateSocketPath("https")

	os.Setenv("CANDY_HTTP_ADDR", "unix/"+httpUnixSocketPath)
	os.Setenv("CANDY_HTTPS_ADDR", "unix/"+httpsUnixSocketPath)

	cfgs := []launchd.SocketProxyConfig{
		{
			LaunchdSocketName: "Socket",
			UnixSocketPath:    httpUnixSocketPath,
		},
		{
			LaunchdSocketName: "SocketTLS",
			UnixSocketPath:    httpsUnixSocketPath,
		},
	}

	var g run.Group
	for _, cfg := range cfgs {
		proxies, err := launchd.NewSocketProxy(cfg)
		if err != nil {
			return err
		}

		for _, proxy := range proxies {
			g.Add(func() error {
				return proxy.Run()
			}, func(err error) {
				proxy.Close()
			})
		}
	}

	{
		g.Add(func() error {
			return startServer(c)
		}, func(err error) {
		})
	}

	return g.Run()
}

func generateSocketPath(name string) string {
	filename := fmt.Sprintf("candy-%s.sock-%d", name, os.Getpid())
	return filepath.Join(os.TempDir(), filename)
}
