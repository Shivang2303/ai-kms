package telemetry

import (
	"context"
	"fmt"
	"log"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/jaeger"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.24.0"
)

/*
LEARNING: JAEGER INTEGRATION FOR DISTRIBUTED TRACING

Jaeger is a distributed tracing system originally developed by Uber.
It helps you:
1. Visualize request flows through your system
2. Find performance bottlenecks
3. Debug errors with full context
4. Analyze service dependencies

Architecture:
  Your App → OpenTelemetry SDK → Jaeger Exporter → Jaeger Collector → Jaeger UI

OpenTelemetry is vendor-neutral, so you can swap Jaeger for other backends
(like Zipkin, Datadog, New Relic) without changing your code!
*/

// InitJaeger initializes Jaeger tracing exporter
// Returns a cleanup function that should be called on shutdown
func InitJaeger(serviceName, jaegerEndpoint string) (func(context.Context) error, error) {
	// Create Jaeger exporter
	// Learning: This sends traces to Jaeger collector
	exp, err := jaeger.New(
		jaeger.WithCollectorEndpoint(jaeger.WithEndpoint(jaegerEndpoint)),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create Jaeger exporter: %w", err)
	}

	// Create resource with service information
	// Learning: Resource identifies your service in Jaeger UI
	res, err := resource.Merge(
		resource.Default(),
		resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceName(serviceName),
			semconv.ServiceVersion("1.0.0"),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create resource: %w", err)
	}

	// Create trace provider with Jaeger exporter
	// Learning: TracerProvider is the central point for creating tracers
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exp), // Batch spans for efficiency
		sdktrace.WithResource(res),
		sdktrace.WithSampler(sdktrace.AlwaysSample()), // Sample 100% of traces (adjust for production)
	)

	// Set global tracer provider
	// Learning: This makes the tracer available throughout your app
	otel.SetTracerProvider(tp)

	log.Printf("✓ Jaeger tracing initialized: %s", jaegerEndpoint)
	log.Printf("  View traces at: http://localhost:16686 (Jaeger UI)")

	// Return cleanup function
	// Learning: Always flush traces on shutdown!
	return tp.Shutdown, nil
}

/*
SAMPLING STRATEGIES (Production Considerations)

1. AlwaysSample() - Sample 100% of traces
   - ✅ Good for: Development, debugging
   - ❌ Bad for: High-traffic production (expensive!)

2. TraceIDRatioBased(0.1) - Sample 10% of traces
   - ✅ Good for: Production with high traffic
   - ⚠️  May miss rare errors

3. ParentBased(AlwaysSample()) - Follow parent's sampling decision
   - ✅ Good for: Microservices (consistent across services)

Example production config:
  sdktrace.WithSampler(
      sdktrace.ParentBased(
          sdktrace.TraceIDRatioBased(0.1), // 10% sampling
      ),
  )
*/
