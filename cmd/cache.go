package cmd

import (
	"fmt"
	"log/slog"

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
	RunE: func(cmd *cobra.Command, args []string) error {
		dir, _ := cache.Dir()
		if err := cache.Clear(); err != nil {
			return fmt.Errorf("failed to clear cache: %w", err)
		}
		slog.Info("cache cleared", "dir", dir)
		return nil
	},
}

var cacheListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all cached parquet query results",
	RunE: func(cmd *cobra.Command, args []string) error {
		entries, err := cache.List()
		if err != nil {
			return fmt.Errorf("failed to list cache: %w", err)
		}

		if len(entries) == 0 {
			fmt.Println("No cached queries.")
			return nil
		}

		for _, e := range entries {
			sizeMB := float64(e.Size) / (1024 * 1024)
			fmt.Printf("[%s] %.1f MB\n  %s\n\n", e.Hash, sizeMB, e.Query)
		}
		return nil
	},
}

var cacheDirCmd = &cobra.Command{
	Use:   "dir",
	Short: "Print the cache directory path",
	RunE: func(cmd *cobra.Command, args []string) error {
		dir, err := cache.Dir()
		if err != nil {
			return err
		}
		fmt.Println(dir)
		return nil
	},
}
