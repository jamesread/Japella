package commands

import (
	"fmt"
	"os"

	"github.com/jamesread/japella/internal/runtimeconfig"
	"github.com/jamesread/japella/internal/utils"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "japella",
	Short: "Japella social media management",
	Long:  "Japella is a self-hosted social media posting and management tool.",
	Run:   runServer,
}

func ExecuteRoot() {
	utils.SetupLogging()
	cobra.OnInitialize(initRuntimeConfig)

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func initRuntimeConfig() {
	runtimeconfig.Get()
}
