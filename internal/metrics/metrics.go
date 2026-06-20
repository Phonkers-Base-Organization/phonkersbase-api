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

// durationBucketBoundaries are second-scale buckets for sub-second API/DB
// latencies. The OTel SDK's default boundaries (5, 10, 25, ...) are
// unit-agnostic and tuned for millisecond-scale values, so they give almost
// no resolution for the second-scale durations recorded here.
var durationBucketBoundaries = []float64{0.005, 0.01, 0.025, 0.05, 0.075, 0.1, 0.25, 0.5, 0.75, 1, 2.5, 5}

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

// RecordDBQuery records the duration of a completed database query.
func RecordDBQuery(ctx context.Context, operation string, d time.Duration, err error) {
	dbQueryDuration.Record(ctx, d.Seconds(),
		metric.WithAttributes(
			attribute.String("db.operation.name", operation),
			attribute.Bool("error", err != nil),
		),
	)
}
