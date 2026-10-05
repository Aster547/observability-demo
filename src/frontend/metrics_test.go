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
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gorilla/mux"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func TestPrometheusMetrics(t *testing.T) {
	r := mux.NewRouter()
	r.Use(prometheusMiddleware)

	r.Handle("/metrics", promhttp.Handler()).Methods(http.MethodGet)
	r.HandleFunc("/_healthz", func(w http.ResponseWriter, _ *http.Request) { fmt.Fprint(w, "ok") })
	r.HandleFunc("/product/{id}", func(w http.ResponseWriter, req *http.Request) {
		vars := mux.Vars(req)
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, "product: %s", vars["id"])
	}).Methods(http.MethodGet)
	r.HandleFunc("/error-endpoint", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}).Methods(http.MethodGet)

	// 1. Send request to /product/0PUK6V6EV0
	req := httptest.NewRequest(http.MethodGet, "/product/0PUK6V6EV0", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rr.Code)
	}

	// 2. Send request to error endpoint
	reqErr := httptest.NewRequest(http.MethodGet, "/error-endpoint", nil)
	rrErr := httptest.NewRecorder()
	r.ServeHTTP(rrErr, reqErr)
	if rrErr.Code != http.StatusInternalServerError {
		t.Fatalf("expected status 500, got %d", rrErr.Code)
	}

	// 3. Send request to /_healthz
	reqHealth := httptest.NewRequest(http.MethodGet, "/_healthz", nil)
	rrHealth := httptest.NewRecorder()
	r.ServeHTTP(rrHealth, reqHealth)
	if rrHealth.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rrHealth.Code)
	}

	// 4. Scrape /metrics
	reqMetrics := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	rrMetrics := httptest.NewRecorder()
	r.ServeHTTP(rrMetrics, reqMetrics)

	if rrMetrics.Code != http.StatusOK {
		t.Fatalf("expected /metrics status 200, got %d", rrMetrics.Code)
	}

	body, err := io.ReadAll(rrMetrics.Body)
	if err != nil {
		t.Fatalf("failed to read /metrics body: %v", err)
	}
	metricsOutput := string(body)
	t.Logf("\n--- Scraped /metrics Output ---\n%s\n", metricsOutput)

	// Verify histogram metrics exist
	if !strings.Contains(metricsOutput, "http_request_duration_seconds_bucket") {
		t.Errorf("expected metric 'http_request_duration_seconds_bucket' in /metrics output")
	}
	if !strings.Contains(metricsOutput, "http_request_duration_seconds_count") {
		t.Errorf("expected metric 'http_request_duration_seconds_count' in /metrics output")
	}
	if !strings.Contains(metricsOutput, "http_requests_total") {
		t.Errorf("expected metric 'http_requests_total' in /metrics output")
	}

	// Verify service="frontend" label exists
	if !strings.Contains(metricsOutput, `service="frontend"`) {
		t.Errorf("expected label service=\"frontend\" in /metrics output")
	}

	// Verify parameterized route template is used rather than the dynamic ID
	if !strings.Contains(metricsOutput, `route="/product/{id}"`) {
		t.Errorf("expected route template '/product/{id}' in labels, output was:\n%s", metricsOutput)
	}
	if strings.Contains(metricsOutput, "0PUK6V6EV0") {
		t.Errorf("cardinality leak detected: raw product ID '0PUK6V6EV0' found in /metrics output")
	}

	// Verify status codes and methods are correctly labelled
	if !strings.Contains(metricsOutput, `method="GET"`) {
		t.Errorf("expected method label 'GET' in /metrics output")
	}
	if !strings.Contains(metricsOutput, `status_code="200"`) {
		t.Errorf("expected status_code='200' in /metrics output")
	}
	if !strings.Contains(metricsOutput, `status_code="500"`) {
		t.Errorf("expected status_code='500' in /metrics output")
	}

	// Verify /_healthz and /metrics are excluded from the histogram
	if strings.Contains(metricsOutput, `route="/_healthz"`) {
		t.Errorf("expected '/_healthz' to be excluded from latency metrics")
	}
	if strings.Contains(metricsOutput, `route="/metrics"`) {
		t.Errorf("expected '/metrics' to be excluded from latency metrics")
	}

	// Verify standard Go runtime metrics are also exposed
	if !strings.Contains(metricsOutput, "go_goroutines") {
		t.Errorf("expected Go runtime metric 'go_goroutines' in /metrics output")
	}
}
