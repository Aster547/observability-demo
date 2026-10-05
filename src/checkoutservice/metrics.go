package main

import (
	"net/http"
	"os"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/sirupsen/logrus"
)

var (
	// orderSuccessesTotal is a counter that counts the number of successfully handled orders
	orderSuccessesTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "grpc_server_handled_total",
			Help: "Total number of successfully handled gRPC orders.",
		},
		[]string{"grpc_service"},
	)
)

// startMetricsServer starts a Prometheus metrics server in a new goroutine.
func startMetricsServer(log logrus.FieldLogger, defaultPort string) {
	metricsPort := defaultPort

	// get metrics port from environment variable or use default port
	if val := os.Getenv("METRICS_PORT"); val != "" {
		metricsPort = val
	}

	// create a new mux and register the /metrics endpoint to serve prometheus metrics
	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.Handler())

	// start the metrics server in a new goroutine
	go func() {
		if log != nil {
			// log the metrics server port
			log.Infof("Prometheus metrics server listening on :%s", metricsPort)
		}
		// listen and serve the metrics server
		if err := http.ListenAndServe(":"+metricsPort, mux); err != nil && err != http.ErrServerClosed {
			if log != nil {
				log.Warnf("metrics server exited: %v", err)
			}
		}
	}()
}
