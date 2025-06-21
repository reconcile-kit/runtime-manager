package runtimemanager

import (
	"github.com/reconcile-kit/controlloop"
	"net/http"
)

type Options struct {
	Logger     controlloop.Logger
	httpClient *http.Client
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
