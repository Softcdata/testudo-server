package main

import (
	"os"
	_ "time/tzdata"

	"github.com/softcdata/testudo-server/cmd"
)

func main() {
	rootCmd := cmd.NewRootCommand()
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
