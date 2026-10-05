package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	pb "github.com/GoogleCloudPlatform/microservices-demo/src/shippingservice/genproto"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	dto "github.com/prometheus/client_model/go"
)

func TestShippingMetrics(t *testing.T) {
	s := &server{}

	req := &pb.ShipOrderRequest{
		Address: &pb.Address{
			StreetAddress: "1600 Amphitheatre Pkwy",
			City:          "Mountain View",
			State:         "CA",
			Country:       "USA",
		},
		Items: []*pb.CartItem{
			{
				ProductId: "OLJCESPC7Z",
				Quantity:  1,
			},
		},
	}

	// 1. Initial count
	var initialCount uint64
	var metric dto.Metric
	if m, ok := shippingHandoffDuration.WithLabelValues("shippingservice", "ShipOrder").(prometheus.Metric); ok {
		if err := m.Write(&metric); err == nil && metric.Histogram != nil {
			initialCount = metric.Histogram.GetSampleCount()
		}
	}

	// 2. Execute ShipOrder to trigger latency observation
	res, err := s.ShipOrder(context.Background(), req)
	if err != nil {
		t.Fatalf("ShipOrder failed: %v", err)
	}
	if res.TrackingId == "" {
		t.Fatalf("ShipOrder returned empty tracking ID")
	}

	// 3. Verify histogram observation was recorded
	m, ok := shippingHandoffDuration.WithLabelValues("shippingservice", "ShipOrder").(prometheus.Metric)
	if !ok {
		t.Fatalf("expected metric to implement prometheus.Metric")
	}
	if err := m.Write(&metric); err != nil {
		t.Fatalf("failed to read histogram metric: %v", err)
	}
	if got := metric.Histogram.GetSampleCount(); got != initialCount+1 {
		t.Errorf("expected sample count %d, got %d", initialCount+1, got)
	}

	// 4. Verify Prometheus /metrics endpoint exposes the histogram metric
	httpReq := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	rr := httptest.NewRecorder()
	handler := promhttp.Handler()
	handler.ServeHTTP(rr, httpReq)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected HTTP status 200 from /metrics, got %d", rr.Code)
	}

	body := rr.Body.String()
	if !strings.Contains(body, "grpc_server_handling_seconds_bucket") {
		t.Errorf("expected /metrics output to contain 'grpc_server_handling_seconds_bucket', got:\n%s", body)
	}
	if !strings.Contains(body, `grpc_service="shippingservice"`) {
		t.Errorf("expected /metrics output to contain 'grpc_service=\"shippingservice\"', got:\n%s", body)
	}
	if !strings.Contains(body, `grpc_method="ShipOrder"`) {
		t.Errorf("expected /metrics output to contain 'grpc_method=\"ShipOrder\"', got:\n%s", body)
	}
}
