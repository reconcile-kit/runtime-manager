package runtimemanager

import (
	"github.com/reconcile-kit/controlloop"
	"github.com/reconcile-kit/controlloop/metrics"
	"go.opentelemetry.io/otel/metric"
	"k8s.io/client-go/util/workqueue"
	"net/http"
)

type Options struct {
	Logger                  controlloop.Logger
	httpClient              *http.Client
	informerAuthConfig      *InformerAuthConfig
	meterProvider           metric.MeterProvider
	controlLopMeterProvider metrics.MetricsProvider
	workQueueMeterProvider  workqueue.MetricsProvider
}

type InformerAuthConfig struct {
	Username  string
	Password  string
	EnableTLS bool
}

type Option func(*Options)

func WithLogger(logger controlloop.Logger) Option {
	return func(o *Options) {
		o.Logger = logger
	}
}

func WithHTTPClient(httpClient *http.Client) Option {
	return func(o *Options) {
		o.httpClient = httpClient
	}
}

func WithInformerAuthConfig(informerAuthConfig *InformerAuthConfig) Option {
	return func(o *Options) {
		o.informerAuthConfig = informerAuthConfig
	}
}

func WithMeterProvider(meterProvider metric.MeterProvider) Option {
	return func(o *Options) {
		o.meterProvider = meterProvider
	}
}

func WithControllerMeterProvider(meterProvider metric.MeterProvider) Option {
	return func(o *Options) {
		o.meterProvider = meterProvider
	}
}

func WithControlLopMeterProvider(controllopMeterProvider metrics.MetricsProvider) Option {
	return func(o *Options) {
		o.controlLopMeterProvider = controllopMeterProvider
	}
}

func WithWorkQueueMeterProvider(workqueueMeterProvider workqueue.MetricsProvider) Option {
	return func(o *Options) {
		o.workQueueMeterProvider = workqueueMeterProvider
	}
}
