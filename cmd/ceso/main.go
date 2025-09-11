package main

import (
	"flag"
	"fmt"
	"net/http"
	"os"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/r11/esxi-commander/pkg/cli"
	"github.com/r11/esxi-commander/pkg/logger"
)

var (
	metricsPort = flag.Int("metrics-port", 0, "Port to expose Prometheus metrics (0 to disable)")
)

func main() {
	flag.Parse()
	logger.Init()

	// Start metrics server if port specified
	if *metricsPort > 0 {
		go func() {
			addr := fmt.Sprintf(":%d", *metricsPort)
			fmt.Printf("Starting metrics server on %s/metrics\n", addr)
			http.Handle("/metrics", promhttp.Handler())
			if err := http.ListenAndServe(addr, nil); err != nil {
				fmt.Printf("Failed to start metrics server: %v\n", err)
			}
		}()
	}

	if err := cli.Execute(); err != nil {
		os.Exit(1)
	}
}
