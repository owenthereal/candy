package cmd

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/owenthereal/candy"
	"github.com/owenthereal/candy/server"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	"go.uber.org/zap"
)

const (
	defaultDNSAddr = "127.0.0.1:25353"
)

var (
	defaultDomains = []string{"test"}
)

var runCmd = &cobra.Command{
	Use:   "run",
	Short: "Starts the Candy process and blocks indefinitely",
	RunE:  runRunE,
}

func init() {
	rootCmd.AddCommand(runCmd)
	addServerFlags(runCmd)
}

func addServerFlags(cmd *cobra.Command) {
	cmd.Flags().String("host-root", filepath.Join(userHomeDir(), ".candy"), "Path to the directory containing applications that will be served by Candy")
	cmd.Flags().StringSlice("domain", defaultDomains, "The top-level domains for which Candy will respond to DNS queries")
	cmd.Flags().String("http-addr", ":28080", "The Proxy server HTTP address")
	cmd.Flags().String("https-addr", ":28443", "The Proxy server HTTPS address")
	cmd.Flags().String("admin-addr", "127.0.0.1:22019", "The Proxy server administrative address")
	cmd.Flags().String("dns-addr", defaultDNSAddr, "The DNS server address")
	cmd.Flags().Bool("dns-local-ip", false, "DNS server responds DNS queries with local IP instead of 127.0.0.1")
}

func runRunE(c *cobra.Command, args []string) error {
	return startServer(c, context.Background())
}

func startServer(c *cobra.Command, ctx context.Context) error {
	cfg, err := loadServerConfig(c)
	if err != nil {
		return err
	}

	candy.Log().Info("using config", zap.Any("cfg", cfg))

	if err := os.MkdirAll(cfg.HostRoot, 0o0755); err != nil {
		return fmt.Errorf("failed to create host directory %s: %w", cfg.HostRoot, err)
	}

	svr := server.New(*cfg)

	return svr.Run(ctx)
}

func loadServerConfig(cmd *cobra.Command) (*server.Config, error) {
	var cfg server.Config

	if err := unmarshalFlags(flagConfigFile, cmd, &cfg); err != nil {
		return nil, err
	}

	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	return &cfg, nil
}

func unmarshalFlags(cfgFile string, cmd *cobra.Command, opts interface{}) error {
	v := viper.New()

	cmd.Flags().VisitAll(func(flag *pflag.Flag) {
		flagName := flag.Name
		if flagName != "config" && flagName != "help" {
			if err := v.BindPFlag(flagName, flag); err != nil {
				panic(fmt.Errorf("error binding flag '%s': %w", flagName, err).Error())
			}
		}
	})

	v.AutomaticEnv()
	v.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))
	v.SetEnvPrefix("CANDY")

	if _, err := os.Stat(cfgFile); err == nil {
		v.SetConfigFile(cfgFile)
		v.SetConfigType("json")
	}

	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return fmt.Errorf("error loading config file %s: %w", cfgFile, err)
		}
	}

	return v.Unmarshal(opts)
}
