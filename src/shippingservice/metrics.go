package main

import (
	"net/http"
	"os"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	shippingHandoffDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "grpc_server_handling_seconds",
			Help:    "Handoff latency for shipping order requests in seconds.",
			Buckets: []float64{0.001, 0.002, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1.0, 2.5},
		},
		[]string{"grpc_service", "grpc_method"},
	)
)

func startMetricsServer(defaultPort string) {
	// get metrics port from environment variable or use default port
	metricsPort := defaultPort
	if val := os.Getenv("METRICS_PORT"); val != "" {
		metricsPort = val
	}

	// create a new mux and register the /metrics endpoint to serve prometheus metrics
	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.Handler())

	// start the metrics server in a new goroutine
	go func() {
		// log is defined in main.go
		if log != nil {
			log.Infof("Prometheus metrics server listening on :%s", metricsPort)
		}
		if err := http.ListenAndServe(":"+metricsPort, mux); err != nil && err != http.ErrServerClosed {
			if log != nil {
				log.Warnf("metrics server exited: %v", err)
			}
		}
	}()
}
