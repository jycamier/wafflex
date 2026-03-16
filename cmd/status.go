package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/jycamier/wafflex/internal/baseline"
	"github.com/jycamier/wafflex/internal/hash"
	"github.com/spf13/cobra"
)

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show current analysis and baseline status",
	Run:   runStatus,
}

func runStatus(cmd *cobra.Command, args []string) {
	resultsDir := resolveResultsDir()

	fmt.Println("Wafflex Status")
	fmt.Println("──────────────")

	// Config
	if appConfig == nil {
		fmt.Println("\nConfig:    (none)")
	} else {
		fmt.Printf("\nConfig:    %s\n", appConfig.CorazaConfig)
		fmt.Printf("Traffic:   %s\n", appConfig.Traffic.Type)
		fmt.Printf("Results:   %s\n", resultsDir)
	}

	// Traffic hash
	if appConfig != nil {
		trafficHash, err := hash.TrafficSourceHash(appConfig)
		if err == nil {
			fmt.Printf("Traffic #: %s\n", trafficHash)

			// Latest result
			latest, _ := hash.FindLatestResult(resultsDir, trafficHash)
			if latest != "" {
				info, _ := os.Stat(latest)
				age := ""
				if info != nil {
					age = fmt.Sprintf(" (%s ago)", time.Since(info.ModTime()).Truncate(time.Second))
				}
				fmt.Printf("\nLatest:    %s%s\n", filepath.Base(latest), age)
			} else {
				fmt.Println("\nLatest:    (no analysis found)")
			}

			// All results for this traffic
			results, _ := hash.FindResults(resultsDir, trafficHash)
			if len(results) > 0 {
				fmt.Printf("Results:   %d file(s)\n", len(results))
			}
		}
	}

	// Baseline
	cur, _ := baseline.Get(resultsDir)
	if cur != "" {
		path, err := baseline.GetPath(resultsDir)
		if err != nil {
			fmt.Printf("\nBaseline:  %s (missing!)\n", cur)
		} else {
			info, _ := os.Stat(path)
			age := ""
			if info != nil {
				age = fmt.Sprintf(" (%s ago)", time.Since(info.ModTime()).Truncate(time.Second))
			}
			fmt.Printf("Baseline:  %s%s\n", cur, age)
		}
	} else {
		fmt.Println("Baseline:  (not set)")
	}

	// Cache
	cacheDir, _ := os.UserHomeDir()
	if cacheDir != "" {
		cacheDir = filepath.Join(cacheDir, ".wafflex", "cache")
		if entries, err := filepath.Glob(filepath.Join(cacheDir, "*.parquet")); err == nil && len(entries) > 0 {
			var totalSize int64
			for _, e := range entries {
				if info, err := os.Stat(e); err == nil {
					totalSize += info.Size()
				}
			}
			fmt.Printf("\nCache:     %d entries (%.1f MB)\n", len(entries), float64(totalSize)/(1024*1024))
		}
	}

	// TTL
	ttl := parseCacheTTL(appConfig)
	if ttl > 0 {
		fmt.Printf("Cache TTL: %s\n", ttl)
	} else {
		fmt.Println("Cache TTL: disabled")
	}
}
