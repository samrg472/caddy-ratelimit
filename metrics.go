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
	"strconv"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// rateLimitMetrics holds all the rate limit metrics
type rateLimitMetrics struct {
	declinedTotal *prometheus.CounterVec
	requestsTotal *prometheus.CounterVec
	processTime   *prometheus.HistogramVec
	keysTotal     *prometheus.GaugeVec
	config        *prometheus.CounterVec
}

var (
	// Metrics registration sync to ensure we only register once
	metricsOnce sync.Once
	// Global metrics instance
	globalMetrics *rateLimitMetrics
)

// initializeMetrics creates and registers all rate limit metrics with Caddy's internal registry
func initializeMetrics(registry prometheus.Registerer) *rateLimitMetrics {
	const ns, sub = "caddy", "rate_limit"

	factory := promauto.With(registry)

	return &rateLimitMetrics{
		// rate_limit_declined_requests_total - Total number of requests declined with HTTP 429
		declinedTotal: factory.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: ns,
				Subsystem: sub,
				Name:      "declined_requests_total",
				Help:      "Total number of requests for which rate limit was applied (Declined with HTTP 429 status code returned).",
			},
			[]string{"zone", "key"},
		),

		// rate_limit_requests_total - Total number of requests that passed through the Rate Limit module
		requestsTotal: factory.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: ns,
				Subsystem: sub,
				Name:      "requests_total",
				Help:      "Total number of requests that passed through Rate Limit module (both declined & processed).",
			},
			[]string{"has_zone"},
		),

		// rate_limit_process_time_seconds - Time taken to process rate limiting for each request
		processTime: factory.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: ns,
				Subsystem: sub,
				Name:      "process_time_seconds",
				Help:      "A time taken to process rate limiting for each request.",
				Buckets:   []float64{.001, .005, .01, .025, .05, .1, .25, .5, 1},
			},
			[]string{"has_zone"},
		),

		// rate_limit_keys_total - Total number of keys that each RL zone contains
		keysTotal: factory.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace: ns,
				Subsystem: sub,
				Name:      "keys_total",
				Help:      "Total number of keys that each RL zone contains. (This metric is collected in the background for each zone.)",
			},
			[]string{"zone"},
		),

		// rate_limit_config - Shows configuration of the rate limiter module
		config: factory.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: ns,
				Subsystem: sub,
				Name:      "config",
				Help:      "Shows configuration of the rate limiter module. Reported only once on bootstrap as configuration is not dynamically configurable.",
			},
			[]string{"zone", "max_events", "window"},
		),
	}
}

// registerMetrics registers all rate limit metrics with the provided Prometheus registry
func registerMetrics(reg prometheus.Registerer) error {
	var err error
	metricsOnce.Do(func() {
		globalMetrics = initializeMetrics(reg)
	})
	return err
}

// metricsCollector holds the metrics collection methods
type metricsCollector struct {
	enabled bool
}

// newMetricsCollector creates a new metrics collector
func newMetricsCollector() *metricsCollector {
	return &metricsCollector{enabled: true}
}

// recordRequest records a request that passed through the rate limit module
func (mc *metricsCollector) recordRequest(hasZone bool) {
	if !mc.enabled || globalMetrics == nil {
		return
	}

	hasZoneStr := "false"
	if hasZone {
		hasZoneStr = "true"
	}
	globalMetrics.requestsTotal.WithLabelValues(hasZoneStr).Inc()
}

// recordDeclinedRequest records a request that was declined due to rate limiting
func (mc *metricsCollector) recordDeclinedRequest(zone, key string) {
	if !mc.enabled || globalMetrics == nil {
		return
	}

	globalMetrics.declinedTotal.WithLabelValues(zone, key).Inc()
}

// recordProcessTime records the time taken to process rate limiting
func (mc *metricsCollector) recordProcessTime(duration time.Duration, hasZone bool) {
	if !mc.enabled || globalMetrics == nil {
		return
	}

	hasZoneStr := "false"
	if hasZone {
		hasZoneStr = "true"
	}
	globalMetrics.processTime.WithLabelValues(hasZoneStr).Observe(duration.Seconds())
}

// updateKeysCount updates the count of keys for a specific zone
func (mc *metricsCollector) updateKeysCount(zone string, count int) {
	if !mc.enabled || globalMetrics == nil {
		return
	}

	globalMetrics.keysTotal.WithLabelValues(zone).Set(float64(count))
}

// recordConfig records the configuration of a rate limit zone (called once during provision)
func (mc *metricsCollector) recordConfig(zone string, maxEvents int, window time.Duration) {
	if !mc.enabled || globalMetrics == nil {
		return
	}

	globalMetrics.config.WithLabelValues(zone,
		strconv.Itoa(maxEvents),
		window.String()).Inc()
}
