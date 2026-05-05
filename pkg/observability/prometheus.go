/*
 * Copyright 2025 - 2026 Zigflow authors <https://github.com/zigflow/zigflow/graphs/contributors>
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package observability

import (
	"github.com/prometheus/client_golang/prometheus"
)

const (
	metricNamespace = "zigflow"
	metricSubsystem = "events"
	labelEmitter    = "emitter"
)

var (
	EventsEmittedTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: metricNamespace,
			Subsystem: metricSubsystem,
			Name:      "emitted_total",
			Help:      "Total number of CloudEvents emitted",
		},
		[]string{labelEmitter, "type"},
	)

	EventsUndeliveredTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: metricNamespace,
			Subsystem: metricSubsystem,
			Name:      "undelivered_total",
			Help:      "Total number of CloudEvents that failed delivery",
		},
		[]string{labelEmitter, "type"},
	)

	EventEmitDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: metricNamespace,
			Subsystem: metricSubsystem,
			Name:      "emit_duration_seconds",
			Help:      "Time taken to emit CloudEvents",
			Buckets:   prometheus.DefBuckets,
		},
		[]string{labelEmitter},
	)
)

func init() {
	prometheus.MustRegister(
		EventsEmittedTotal,
		EventsUndeliveredTotal,
		EventEmitDuration,
	)
}
