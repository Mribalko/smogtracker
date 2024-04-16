package trace

import (
	"context"
	"errors"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.21.0"
)

type (
	traceOptions struct {
		attrs       []attribute.KeyValue
		otelGrpcURL string
	}
	Option func(*traceOptions) error
)

// Initialises OTEL tracing, registers global TracerProvider,
func New(ctx context.Context, options ...Option) (*sdktrace.TracerProvider, error) {

	tops := &traceOptions{}
	for _, opt := range options {
		if err := opt(tops); err != nil {
			return nil, err
		}
	}

	var (
		exporter sdktrace.SpanExporter
		err      error
	)
	if len(tops.otelGrpcURL) != 0 {
		exporter, err = otlptracegrpc.New(
			ctx,
			otlptracegrpc.WithEndpoint(tops.otelGrpcURL),
			otlptracegrpc.WithInsecure(),
		)
	} else {
		exporter, err = stdouttrace.New(stdouttrace.WithPrettyPrint())
	}
	if err != nil {
		return nil, err
	}

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(resource.NewWithAttributes(
			semconv.SchemaURL,
			tops.attrs...,
		)),
	)

	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(propagation.TraceContext{}, propagation.Baggage{}))

	return tp, nil

}

func WithServiceName(serviceName string) Option {
	return func(tops *traceOptions) error {
		if len(serviceName) == 0 {
			return errors.New("service name is empty")
		}
		tops.attrs = append(tops.attrs, semconv.ServiceNameKey.String(serviceName))
		return nil
	}
}

func WithServiceVersion(serviceVersion string) Option {
	return func(tops *traceOptions) error {
		if len(serviceVersion) == 0 {
			return errors.New("service version is empty")
		}
		tops.attrs = append(tops.attrs, semconv.ServiceVersion(serviceVersion))
		return nil
	}
}

func WithDeploymentEnv(deploymentEnv string) Option {
	return func(tops *traceOptions) error {
		if len(deploymentEnv) == 0 {
			return errors.New("deploymentEnv is empty")
		}
		tops.attrs = append(tops.attrs, semconv.DeploymentEnvironmentKey.String(deploymentEnv))
		return nil
	}
}

func WithOtelGrpcURL(otelGrpcURL string) Option {
	return func(tops *traceOptions) error {
		if len(otelGrpcURL) == 0 {
			return errors.New("otelGrpcURL is empty")
		}
		tops.otelGrpcURL = otelGrpcURL
		return nil
	}
}
