// Package metrics wires up OpenTelemetry metrics for the API.
//
// Like logging, metrics are a cross-cutting concern: instruments are
// package-level and initialized once at startup via Init. Callers never
// receive or pass around a metrics struct.
package metrics

import (
	"context"
	"time"

	"go.opentelemetry.io/contrib/instrumentation/runtime"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp"
	"go.opentelemetry.io/otel/metric"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
)

const meterName = "github.com/PhonkersBase/base-api2"

// durationBucketBoundaries are second-scale buckets for API/DB latencies, weighted
// towards the sub-millisecond end where this service actually operates: a served
// request is ~0.5ms when the search misses and ~2ms when it hits, and an individual
// query is ~0.2-1ms. Boundaries starting at 5ms — the SDK default, and what this
// list used to start at — put all of that in the first bucket, so every quantile
// reads as "fast" and an 8x regression from 0.5ms to 4ms stays invisible.
//
// The top of the range is kept coarse but present: bursts on a cold connection pool
// reach tens of ms, and a pathological multi-term search can still take ~0.5s.
var durationBucketBoundaries = []float64{
	0.0001, 0.00025, 0.0005, 0.001, 0.0025, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5,
}

var (
	httpRequestDuration metric.Float64Histogram
	dbQueryDuration     metric.Float64Histogram
	errorsTotal         metric.Int64Counter
)

func init() {
	meter := otel.GetMeterProvider().Meter(meterName)
	mustInitInstruments(meter)
}

func mustInitInstruments(meter metric.Meter) {
	var err error

	httpRequestDuration, err = meter.Float64Histogram(
		"http.server.request.duration",
		metric.WithDescription("Duration of HTTP requests"),
		metric.WithUnit("s"),
		metric.WithExplicitBucketBoundaries(durationBucketBoundaries...),
	)
	if err != nil {
		panic(err)
	}

	dbQueryDuration, err = meter.Float64Histogram(
		"db.client.query.duration",
		metric.WithDescription("Duration of database queries"),
		metric.WithUnit("s"),
		metric.WithExplicitBucketBoundaries(durationBucketBoundaries...),
	)
	if err != nil {
		panic(err)
	}

	errorsTotal, err = meter.Int64Counter(
		"app.errors",
		metric.WithDescription("Count of error/fatal level log events"),
	)
	if err != nil {
		panic(err)
	}
}

// Init configures the global MeterProvider and re-binds package-level
// instruments to it. If otlpEndpoint is empty, metrics stay bound to the
// SDK's default no-op provider so local dev doesn't need a collector
// configured.
//
// The returned shutdown func flushes and stops the exporter; call it during
// graceful shutdown.
func Init(ctx context.Context, serviceName, otlpEndpoint string) (shutdown func(context.Context) error, err error) {
	if otlpEndpoint == "" {
		return func(context.Context) error { return nil }, nil
	}

	exporter, err := otlpmetrichttp.New(ctx)
	if err != nil {
		return nil, err
	}

	res, err := resource.Merge(resource.Default(), resource.NewSchemaless(
		attribute.String("service.name", serviceName),
	))
	if err != nil {
		return nil, err
	}

	mp := sdkmetric.NewMeterProvider(
		sdkmetric.WithResource(res),
		sdkmetric.WithReader(sdkmetric.NewPeriodicReader(exporter, sdkmetric.WithInterval(15*time.Second))),
	)
	otel.SetMeterProvider(mp)

	mustInitInstruments(mp.Meter(meterName))

	if err := runtime.Start(runtime.WithMeterProvider(mp)); err != nil {
		return nil, err
	}

	return mp.Shutdown, nil
}

// RecordHTTPRequest records the duration of a completed HTTP request.
func RecordHTTPRequest(ctx context.Context, method, route string, status int, d time.Duration) {
	httpRequestDuration.Record(ctx, d.Seconds(),
		metric.WithAttributes(
			attribute.String("http.request.method", method),
			attribute.String("http.route", route),
			attribute.Int("http.response.status_code", status),
		),
	)
}

// RecordDBQuery records the duration of a completed database query. name is
// the low-cardinality query identifier parsed from its "-- name: ..." leading
// comment (see metrics.queryName), or "unknown" if the query isn't tagged.
func RecordDBQuery(ctx context.Context, name, operation string, d time.Duration, err error) {
	dbQueryDuration.Record(ctx, d.Seconds(),
		metric.WithAttributes(
			attribute.String("db.operation.name", operation),
			attribute.String("db.query.name", name),
			attribute.Bool("error", err != nil),
		),
	)
}
