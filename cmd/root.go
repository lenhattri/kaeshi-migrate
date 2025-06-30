package cmd

import (
	"bufio"
	"strings"

	"github.com/spf13/cobra"
)

var (
	yesFlag bool
	rootCmd *cobra.Command
)

// NewRootCmd builds the top-level command with global flags.
func NewRootCmd() *cobra.Command {
	rootCmd = &cobra.Command{
		Use:           "kaeshi",
		Short:         "Database migration manager",
		SilenceUsage:  true,
		SilenceErrors: true,
	}
	rootCmd.PersistentFlags().BoolVarP(&yesFlag, "yes", "y", false, "automatic yes to prompts")
	return rootCmd
}

// askConfirmation prints msg and waits for user to type y/yes.
func AskConfirmation(msg string) (bool, error) {
	if yesFlag {
		return true, nil
	}
	rootCmd.Print(msg + " [y/N]: ")
	reader := bufio.NewReader(rootCmd.InOrStdin())
	line, err := reader.ReadString('\n')
	if err != nil {
		return false, err
	}
	ans := strings.ToLower(strings.TrimSpace(line))
	return ans == "y" || ans == "yes", nil
}
