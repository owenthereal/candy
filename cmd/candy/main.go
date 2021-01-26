package main

import (
	"github.com/owenthereal/candy"
	"github.com/owenthereal/candy/cmd/candy/command"
	"go.uber.org/zap"
)

func main() {
	rootCmd := command.Root()
	if err := rootCmd.Execute(); err != nil {
		candy.Log().Fatal("", zap.Error(err))
	}
}
