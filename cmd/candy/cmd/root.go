package cmd

import (
	"os"
	"os/user"
	"path/filepath"

	"github.com/owenthereal/candy"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

func Execute() error {
	return rootCmd.Execute()
}

var rootCmd = &cobra.Command{
	Use:   "candy",
	Short: "A zero-config reverse proxy server",
}

var (
	homeDir         string
	flagRootCfgFile string
)

func init() {
	rootCmd.PersistentFlags().StringVar(&flagRootCfgFile, "config", filepath.Join(userHomeDir(), ".candyconfig"), "Config file")
}

func userHomeDir() string {
	if homeDir != "" {
		return homeDir
	}

	var (
		sudo = os.Getenv("SUDO_USER")
		euid = os.Geteuid()
		err  error
	)

	if sudo != "" && euid == 0 {
		u, err := user.Lookup(sudo)
		if err != nil {
			candy.Log().Fatal("error looking up sudo user", zap.String("user", sudo), zap.Error(err))
		}

		return u.HomeDir
	}

	homeDir, err = os.UserHomeDir()
	if err != nil {
		candy.Log().Fatal("error getting user home directory", zap.Error(err))
	}

	return homeDir
}
