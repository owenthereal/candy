package main

import (
	"github.com/owenthereal/candy"
	"github.com/owenthereal/candy/cmd/candy/cmd"
	"go.uber.org/zap"
)

func main() {
	if err := cmd.Execute(); err != nil {
		candy.Log().Fatal("", zap.Error(err))
	}
}
