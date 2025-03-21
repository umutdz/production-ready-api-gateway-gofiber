package tracing

import (
	"context"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.4.0"
	"go.uber.org/zap"
	"time"
)

func InitTracer(ctx context.Context, serviceName string, jaegerEndpoint string) (*sdktrace.TracerProvider, func(context.Context) error) {
	var exporter sdktrace.SpanExporter
	var err error


	// Log the endpoint being used
	zap.L().Info("Initializing OTLP gRPC exporter",
		zap.String("endpoint", jaegerEndpoint))

	// gRPC exporter
	exporter, err = otlptracegrpc.New(ctx,
		otlptracegrpc.WithEndpoint(jaegerEndpoint),
		otlptracegrpc.WithInsecure(),
	)
	if err != nil {
		zap.L().Error("Failed to create OTLP gRPC exporter",
			zap.Error(err),
			zap.String("endpoint", jaegerEndpoint))
		return nil, func(context.Context) error { return nil }
	}

	zap.L().Info("Creating trace provider with service name",
		zap.String("service_name", serviceName))

	// Resource
	res := resource.NewWithAttributes(
		semconv.SchemaURL,
		semconv.ServiceNameKey.String(serviceName),
	)

	// Trace provider with more frequent exports for testing
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
		sdktrace.WithBatcher(exporter,
			sdktrace.WithMaxExportBatchSize(10),   // Export smaller batches
			sdktrace.WithBatchTimeout(5 * time.Second),  // Export more frequently
		),
		sdktrace.WithResource(res),
	)

	// Global provider and propagator
	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	zap.L().Info("Tracer initialized successfully",
		zap.String("service", serviceName),
		zap.String("endpoint", jaegerEndpoint))

	return tp, tp.Shutdown
}

