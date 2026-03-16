package cmd

import (
	"fmt"
	"log/slog"
	"os"

	"github.com/jycamier/wafflex/internal/cache"
	"github.com/spf13/cobra"
)

var cacheCmd = &cobra.Command{
	Use:   "cache",
	Short: "Manage the parquet query cache",
}

var cacheClearCmd = &cobra.Command{
	Use:   "clear",
	Short: "Clear all cached parquet query results",
	Run: func(cmd *cobra.Command, args []string) {
		dir, _ := cache.Dir()
		if err := cache.Clear(); err != nil {
			slog.Error("failed to clear cache", "error", err)
			os.Exit(1)
		}
		slog.Info("cache cleared", "dir", dir)
	},
}

var cacheListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all cached parquet query results",
	Run: func(cmd *cobra.Command, args []string) {
		entries, err := cache.List()
		if err != nil {
			slog.Error("failed to list cache", "error", err)
			os.Exit(1)
		}

		if len(entries) == 0 {
			fmt.Println("No cached queries.")
			return
		}

		for _, e := range entries {
			sizeMB := float64(e.Size) / (1024 * 1024)
			fmt.Printf("[%s] %.1f MB\n  %s\n\n", e.Hash, sizeMB, e.Query)
		}
	},
}

var cacheDirCmd = &cobra.Command{
	Use:   "dir",
	Short: "Print the cache directory path",
	Run: func(cmd *cobra.Command, args []string) {
		dir, err := cache.Dir()
		if err != nil {
			slog.Error("failed to get cache directory", "error", err)
			os.Exit(1)
		}
		fmt.Println(dir)
	},
}
