// Copyright 2026 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gorilla/mux"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	httpRequestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_request_duration_seconds",
			Help:    "Latency of HTTP requests in seconds.",
			Buckets: []float64{0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1.0, 2.5, 5.0, 10.0},
		},
		[]string{"service", "method", "route", "status_code"},
	)

	httpRequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "Total count of HTTP requests processed by the frontend.",
		},
		[]string{"service", "method", "route", "status_code"},
	)
)

func prometheusMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()           // record start time
		rr := &responseRecorder{w: w} // wrapper for the response writer to capture the status code

		next.ServeHTTP(rr, r) // execute handler first and blocks code execution till handler finishes

		var routePattern string

		// Returns the MuxRoute that will be matched for the given request, if any.
		if route := mux.CurrentRoute(r); route != nil {
			if template, err := route.GetPathTemplate(); err == nil {
				routePattern = template
			} else if name := route.GetName(); name != "" {
				routePattern = name
			}
		}
		if routePattern == "" {
			routePattern = "unmatched"
		}

		// Skip recording metrics for internal endpoints
		if routePattern == baseUrl+"/metrics" || routePattern == baseUrl+"/_healthz" || routePattern == baseUrl+"/robots.txt" {
			return
		}

		// if status is 0, it means the handler didn't write any status code, so we default to 200 OK
		status := rr.status
		if status == 0 {
			status = http.StatusOK
		}

		// convert status code to string
		statusCodeStr := strconv.Itoa(status)

		// get duration in seconds
		duration := time.Since(start).Seconds()

		httpRequestDuration.WithLabelValues("frontend", r.Method, routePattern, statusCodeStr).Observe(duration)
		httpRequestsTotal.WithLabelValues("frontend", r.Method, routePattern, statusCodeStr).Inc()
	})
}
