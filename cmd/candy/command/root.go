package command

import (
	"os"
	"path/filepath"

	"github.com/owenthereal/candy"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

var (
	homeDir         string
	flagRootCfgFile string
)

func init() {
	var err error
	homeDir, err = os.UserHomeDir()
	if err != nil {
		candy.Log().Fatal("error getting home directory", zap.Error(err))
	}
}

func Root() *cobra.Command {
	rootCmd := &cobra.Command{
		Use:   "candy",
		Short: "A zero-config reverse proxy server",
	}

	rootCmd.AddCommand(Run())
	if launch := Launch(); launch != nil {
		rootCmd.AddCommand(launch)
	}
	if setup := Setup(); setup != nil {
		rootCmd.AddCommand(setup)
	}

	rootCmd.PersistentFlags().StringVar(&flagRootCfgFile, "config", filepath.Join(homeDir, ".candyconfig"), "Config file")

	return rootCmd
}
