package command

import (
	"github.com/spf13/cobra"
)

func Setup() *cobra.Command {
	return newSetupCmd()
}
