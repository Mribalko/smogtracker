package metric

import (
	"context"
	"errors"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/prometheus"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.21.0"
)

type (
	traceOptions struct {
		attrs []attribute.KeyValue
	}
	Option func(*traceOptions) error
)

// Initialises OTEL metrics, registers global MeterProvider
func Init(ctx context.Context, options ...Option) error {

	tops := &traceOptions{}
	for _, opt := range options {
		if err := opt(tops); err != nil {
			return err
		}
	}

	metricExporter, err := prometheus.New()
	if err != nil {
		return err
	}

	mp := sdkmetric.NewMeterProvider(
		sdkmetric.WithReader(metricExporter),
		sdkmetric.WithResource(resource.NewWithAttributes(
			semconv.SchemaURL,
			tops.attrs...,
		)),
	)

	otel.SetMeterProvider(mp)

	go func() {
		<-ctx.Done()
		mp.Shutdown(ctx)
	}()

	return nil
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
