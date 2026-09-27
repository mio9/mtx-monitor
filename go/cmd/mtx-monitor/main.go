package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"mio9/mtx-monitor/internal/cli"
	"mio9/mtx-monitor/internal/config"
	"mio9/mtx-monitor/internal/tui"
)

var rootCmd = &cobra.Command{
	Use:   "mtx-monitor",
	Short: "MediaMTX publisher bitrate monitor and dashboard",
	Long: `mtx-monitor polls MediaMTX for publishing sessions, tracks their bitrates,
and can kick over-limit publishers. Runs an interactive TUI by default,
or headless CLI mode with --noui.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.LoadConfig()
		if err != nil {
			return fmt.Errorf("load config: %w", err)
		}

		noUi, _ := cmd.Flags().GetBool("noui")
		if noUi {
			return cli.RunCLI(cfg)
		}
		return tui.RunTUI(*cfg)
	},
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print version information",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("mtx-monitor v0.1.0 (Go)")
	},
}

func init() {
	rootCmd.Flags().BoolP("noui", "", false, "Run in headless CLI mode (no TUI)")
	rootCmd.Flags().BoolP("help", "h", false, "Help for mtx-monitor")
	rootCmd.AddCommand(versionCmd)
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}
