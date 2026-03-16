package cmd

import (
	"fmt"
	"log/slog"
	"os"

	"github.com/jycamier/wafflex/internal/config"
	applog "github.com/jycamier/wafflex/internal/log"
	"github.com/spf13/cobra"
)

var (
	rootCmd = &cobra.Command{
		Use:   "wafflex",
		Short: "Test WAF rules by replaying real traffic",
		Long:  `Wafflex replays HTTP traffic through Coraza WAF and analyzes the results.`,
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			v, _ := cmd.Flags().GetCount("verbosity")
			applog.Init(verbosityLevel(v))
		},
	}

	appConfig *config.Config
)

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func verbosityLevel(count int) slog.Level {
	switch {
	case count >= 3:
		return slog.LevelDebug
	case count == 2:
		return slog.LevelInfo
	case count == 1:
		return slog.LevelWarn
	default:
		return slog.LevelInfo
	}
}

func init() {
	cobra.OnInitialize(initConfig)
	rootCmd.PersistentFlags().CountP("verbosity", "v", "Log verbosity: -v (warn), -vv (info), -vvv+ (debug). Default: error only")
	rootCmd.PersistentFlags().String("config", "", "config file (default: .wafflex.yaml in current dir or home)")
	cacheCmd.AddCommand(cacheClearCmd)
	cacheCmd.AddCommand(cacheListCmd)
	cacheCmd.AddCommand(cacheDirCmd)
	rootCmd.AddCommand(analyzeCmd)
	rootCmd.AddCommand(queryCmd)
	rootCmd.AddCommand(exploreCmd)
	rootCmd.AddCommand(diffCmd)
	rootCmd.AddCommand(cacheCmd)
	rootCmd.AddCommand(versionCmd)
}

func initConfig() {
	cfgPath, _ := rootCmd.PersistentFlags().GetString("config")

	if cfgPath != "" {
		cfg, err := config.Load(cfgPath)
		if err != nil {
			slog.Error("failed to load config", "path", cfgPath, "error", err)
			os.Exit(1)
		}
		appConfig = cfg
		return
	}

	cwd, err := os.Getwd()
	if err != nil {
		return
	}

	cfg, err := config.Search(cwd)
	if err != nil {
		slog.Warn("error reading config", "error", err)
		return
	}
	appConfig = cfg
}
