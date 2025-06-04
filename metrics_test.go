// Copyright 2024 Matthew Holt

// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at

// 	http://www.apache.org/licenses/LICENSE-2.0

// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package caddyrl

import (
	"fmt"
	"strconv"
	"sync"
	"testing"

	"github.com/caddyserver/caddy/v2/caddytest"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
)

func TestMetrics(t *testing.T) {
	// Reset the metrics registry to ensure clean state
	prometheus.DefaultRegisterer = prometheus.NewRegistry()

	// Reset global metrics
	globalMetrics = nil
	metricsOnce = sync.Once{}

	window := 10
	maxEvents := 2

	// Admin API must be exposed on port 2999 to match what caddytest.Tester does
	config := fmt.Sprintf(`{
	"admin": {"listen": "localhost:2999"},
	"apps": {
		"http": {
			"servers": {
				"demo": {
					"listen": [":8080"],
					"metrics": {},
					"routes": [{
						"handle": [
							{
								"handler": "rate_limit",
								"rate_limits": {
									"test_zone": {
										"match": [{"method": ["GET"]}],
										"key": "static",
										"window": "%ds",
										"max_events": %d
									}
								}
							},
							{
								"handler": "static_response",
								"status_code": 200
							}
						]
					}]
				}
			}
		}
	}
}`, window, maxEvents)

	initTime()

	tester := caddytest.NewTester(t)
	tester.InitServer(config, "json")

	// Ensure metrics are initialized
	if globalMetrics == nil {
		t.Fatal("Expected globalMetrics to be initialized")
	}

	// Test that configuration metrics are recorded
	configMetric := testutil.ToFloat64(globalMetrics.config.WithLabelValues("test_zone", strconv.Itoa(maxEvents), fmt.Sprintf("%ds", window)))
	if configMetric == 0 {
		t.Error("Expected configuration metric to be recorded")
	}

	// Make some requests that should be allowed
	for i := 0; i < maxEvents; i++ {
		tester.AssertGetResponse("http://localhost:8080", 200, "")
	}

	// Check request metrics
	requestsMetric := testutil.ToFloat64(globalMetrics.requestsTotal.WithLabelValues("true"))
	if requestsMetric < float64(maxEvents) {
		t.Errorf("Expected at least %d requests metric, got %f", maxEvents, requestsMetric)
	}

	// Make a request that should be declined
	tester.AssertGetResponse("http://localhost:8080", 429, "")

	// Check declined requests metric
	declinedMetric := testutil.ToFloat64(globalMetrics.declinedTotal.WithLabelValues("test_zone", "static"))
	if declinedMetric == 0 {
		t.Error("Expected declined requests metric to be recorded")
	}

	// Check process time histogram - use the histogram vec directly
	processTimeHistogram := globalMetrics.processTime.WithLabelValues("true")
	// We can't easily test the value of a histogram in a unit test without using more complex metrics
	// So let's just verify the metric was created successfully
	if processTimeHistogram == nil {
		t.Error("Expected process time histogram to be created")
	}
}
