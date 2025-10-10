package runtimemanager

import (
	"github.com/reconcile-kit/controlloop"
	"go.opentelemetry.io/otel/metric"
	"net/http"
)

type Options struct {
	Logger             controlloop.Logger
	httpClient         *http.Client
	informerAuthConfig *InformerAuthConfig
	meterProvider      metric.Meter
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

func WithMeterProvider(meterProvider metric.Meter) Option {
	return func(o *Options) {
		o.meterProvider = meterProvider
	}
}
