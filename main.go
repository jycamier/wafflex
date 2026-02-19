package main

import (
	"github.com/jycamier/wafflex/cmd"
)

var (
	version = "dev"
	commit  = "none"
)

func main() {
	cmd.SetVersionInfo(version, commit)
	cmd.Execute()
}
