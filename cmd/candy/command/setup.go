package command

import (
	"os"
	"os/exec"

	"github.com/spf13/cobra"
)

func Setup() *cobra.Command {
	return newSetupCmd()
}

func execCmd(c ...string) error {
	cmd := exec.Command(c[0], c[1:]...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}
