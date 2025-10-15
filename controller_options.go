package runtimemanager

import "time"

type controllerOptions struct {
	concurrentWorkers int
	rateLimits        *storageRateLimits
}

type storageRateLimits struct {
	Min time.Duration
	Max time.Duration
}

type ControllerOption func(*controllerOptions)

func WithConcurrentWorkers(workers int) ControllerOption {
	return func(o *controllerOptions) {
		o.concurrentWorkers = workers
	}
}

func WithCustomRateLimits(min, max time.Duration) ControllerOption {
	return func(o *controllerOptions) {
		o.rateLimits = &storageRateLimits{
			Min: min,
			Max: max,
		}
	}
}
