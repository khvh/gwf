package telemetry

import (
	"context"

	"github.com/khvh/gwf/pkg/config"
	"github.com/labstack/echo/v4"
	"github.com/rs/zerolog/log"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/jaeger"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.4.0"
	"go.opentelemetry.io/otel/trace"
)

func New() {
	exporter, err := jaeger.New(jaeger.WithCollectorEndpoint(jaeger.WithEndpoint(config.Get().Telemetry.JaegerEndpoint)))
	if err != nil {
		log.Fatal().Err(err).Send()
	}

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(
			resource.NewWithAttributes(
				semconv.SchemaURL,
				semconv.ServiceNameKey.String(config.Get().ID),
			),
		),
	)

	otel.
		SetTracerProvider(tp)
	otel.
		SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(propagation.TraceContext{}, propagation.Baggage{}))
}

// WithTracer returns a new tracer with name
func WithTracer(name string) trace.Tracer {
	return otel.GetTracerProvider().Tracer(name)
}

// WithSpan returns a new span for tracer
func WithSpan(ctx context.Context, tracer trace.Tracer, name string) (context.Context, trace.Span) {
	return tracer.Start(ctx, name)
}

// WithEchoTracer gets the tracer from Echo's context
func WithEchoTracer(c echo.Context) trace.Tracer {
	return c.Get("otel-go-contrib-tracer-labstack-echo").(trace.Tracer)
}
