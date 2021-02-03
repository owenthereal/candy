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
	var err error
	homeDir, err = userHomeDir()
	if err != nil {
		candy.Log().Fatal("error getting user home directory", zap.Error(err))
	}

	rootCmd.PersistentFlags().StringVar(&flagRootCfgFile, "config", filepath.Join(homeDir, ".candyconfig"), "Config file")
}

func userHomeDir() (string, error) {
	var (
		sudo = os.Getenv("SUDO_USER")
		euid = os.Geteuid()
	)

	if sudo != "" && euid == 0 {
		u, err := user.Lookup(sudo)
		if err != nil {
			return "", nil
		}

		return u.HomeDir, nil
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	return homeDir, nil
}
