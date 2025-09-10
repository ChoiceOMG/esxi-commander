package main

import (
	"os"

	"github.com/r11/esxi-commander/pkg/cli"
	"github.com/r11/esxi-commander/pkg/logger"
)

func main() {
	logger.Init()

	if err := cli.Execute(); err != nil {
		os.Exit(1)
	}
}
