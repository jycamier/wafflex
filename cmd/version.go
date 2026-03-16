package cmd

import (
	"fmt"
	"runtime/debug"

	"github.com/spf13/cobra"
)

var (
	version = "dev"
	commit  = "none"
)

func SetVersionInfo(v, c string) {
	version = v
	commit = c
}

func resolveVersion() (string, string) {
	if version != "dev" {
		return version, commit
	}
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return version, commit
	}
	v := info.Main.Version
	if v != "" && v != "(devel)" {
		version = v
	}
	for _, s := range info.Settings {
		if s.Key == "vcs.revision" && len(s.Value) >= 12 {
			commit = s.Value[:12]
			break
		}
	}
	return version, commit
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print version information",
	Run: func(cmd *cobra.Command, args []string) {
		v, c := resolveVersion()
		fmt.Printf("wafflex %s (commit: %s)\n", v, c)
	},
}
