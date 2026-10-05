package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	dto "github.com/prometheus/client_model/go"
)

func TestCheckoutMetrics(t *testing.T) {
	// 1. Check initial counter value
	var initialVal float64
	var metric dto.Metric

	// get initial value of the counter
	if err := orderSuccessesTotal.WithLabelValues("checkoutservice").Write(&metric); err == nil && metric.Counter != nil {
		initialVal = metric.Counter.GetValue()
	}

	// 2. Increment counter (as executed at the end of PlaceOrder)
	orderSuccessesTotal.WithLabelValues("checkoutservice").Inc()

	// 3. Verify counter value incremented
	if err := orderSuccessesTotal.WithLabelValues("checkoutservice").Write(&metric); err != nil {
		t.Fatalf("failed to read counter metric: %v", err)
	}
	if got := metric.Counter.GetValue(); got != initialVal+1 {
		t.Errorf("expected counter value %f, got %f", initialVal+1, got)
	}

	// 4. Verify Prometheus /metrics endpoint exposes the metric with correct label and value
	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	rr := httptest.NewRecorder()
	handler := promhttp.Handler()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected HTTP status 200 from /metrics, got %d", rr.Code)
	}

	body := rr.Body.String()
	if !strings.Contains(body, "grpc_server_handled_total") {
		t.Errorf("expected /metrics output to contain 'grpc_server_handled_total', got:\n%s", body)
	}
	if !strings.Contains(body, `grpc_service="checkoutservice"`) {
		t.Errorf("expected /metrics output to contain 'grpc_service=\"checkoutservice\"', got:\n%s", body)
	}
}
