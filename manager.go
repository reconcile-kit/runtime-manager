package runtimemanager

import (
	"context"
	"fmt"
	"github.com/reconcile-kit/controlloop/metrics"
	"go.opentelemetry.io/otel"
	"k8s.io/client-go/util/workqueue"
	"net/http"
	"sync"

	"github.com/reconcile-kit/api/resource"
	cl "github.com/reconcile-kit/controlloop"
	event "github.com/reconcile-kit/redis-informer-provider"
	cmetrics "github.com/reconcile-kit/runtime-manager/observability/controller/metrics"
	wmetrics "github.com/reconcile-kit/runtime-manager/observability/workqueue/metrics"
	state "github.com/reconcile-kit/state-manager-provider"
	"go.opentelemetry.io/otel/metric"
)

type InitReconciler[T resource.Object[T]] interface {
	cl.Reconcile[T]
	SetStorage(storage *cl.StorageSet)
}

type Stopped interface {
	Stop()
}

type controller interface {
	Run()
	Stop()
}

type Manager struct {
	shardID                 string
	receivers               []cl.Receiver
	informerAddress         string
	informerAuthConfig      *InformerAuthConfig
	externalStorageAddress  string
	storageSet              *cl.StorageSet
	stopped                 []Stopped
	logger                  cl.Logger
	httpClient              *http.Client
	controllers             []*ctrlWrapper
	meterProvider           metric.Meter
	controllopMeterProvider metrics.MetricsProvider
	workqueueMeterProvider  workqueue.MetricsProvider
}

type ctrlWrapper struct {
	controller controller
	receiver   cl.Receiver
}

func New(shardID string, informerAddr, externalStorageAddr string, opts ...Option) *Manager {
	options := &Options{}
	for _, o := range opts {
		o(options)
	}
	storageSet := cl.NewStorageSet()
	application := &Manager{
		shardID:                shardID,
		informerAddress:        informerAddr,
		externalStorageAddress: externalStorageAddr,
		storageSet:             storageSet,
	}
	if options.Logger != nil {
		application.logger = options.Logger
	} else {
		application.logger = &cl.SimpleLogger{}
	}

	if options.meterProvider != nil {
		application.meterProvider = options.meterProvider
	}

	application.controllopMeterProvider = cmetrics.NewControllerMetrics(otel.Meter("controlloop"))
	if options.controllopMeterProvider != nil {
		application.controllopMeterProvider = options.controllopMeterProvider
	}

	application.workqueueMeterProvider = wmetrics.NewWorkqueueMetricsProvider(otel.Meter("workqueue"))
	if options.workqueueMeterProvider != nil {
		application.workqueueMeterProvider = options.workqueueMeterProvider
	}

	if options.httpClient != nil {
		application.httpClient = options.httpClient
	} else {
		application.httpClient = &http.Client{}
	}

	if options.informerAuthConfig != nil {
		application.informerAuthConfig = options.informerAuthConfig
	}

	application.logger.Info(
		fmt.Sprintf("Shard ID:%s, InformerAddress:%s, ExternalAddress:%s",
			shardID,
			informerAddr,
			externalStorageAddr),
	)
	return application
}

func (a *Manager) Run(ctx context.Context) error {
	redisConfig := &event.RedisConfig{
		Addr: a.informerAddress,
	}
	if a.informerAuthConfig != nil {
		redisConfig.Username = a.informerAuthConfig.Username
		redisConfig.Password = a.informerAuthConfig.Password
		redisConfig.EnableTLS = a.informerAuthConfig.EnableTLS
	}

	var receivers []cl.Receiver
	for _, r := range a.controllers {
		r.controller.Run()
		receivers = append(receivers, r.receiver)
		a.stopped = append(a.stopped, r.controller)
	}

	eventProvider, err := event.NewRedisStreamListenerWithConfig(redisConfig, a.shardID, event.WithLogger(a.logger))
	if err != nil {
		return err
	}

	informer := cl.NewStorageInformer(a.shardID, eventProvider, receivers)
	err = informer.Run(ctx)
	if err != nil {
		return err
	}
	return nil
}

func (a *Manager) Shutdown(ctx context.Context) error {
	a.Stop()
	return nil
}

func (a *Manager) Stop() {
	stoopedCount := len(a.stopped)
	if stoopedCount == 0 {
		return
	}
	wg := &sync.WaitGroup{}
	wg.Add(stoopedCount)
	for _, stopped := range a.stopped {
		go func() {
			defer wg.Done()
			stopped.Stop()
		}()
	}

	wg.Wait()
}

func SetRemoteClient[T resource.Object[T]](manager *Manager) error {
	sm, err := state.NewStateManagerProvider[T](manager.externalStorageAddress, manager.httpClient)
	if err != nil {
		return err
	}
	rc, err := cl.NewRemoteClient[T](sm)
	if err != nil {
		return err
	}
	cl.SetStorage[T](manager.storageSet, rc)
	return nil
}

func SetController[T resource.Object[T]](manager *Manager, controller InitReconciler[T], opts ...cl.StorageOption) error {
	sm, err := state.NewStateManagerProvider[T](manager.externalStorageAddress, manager.httpClient)
	if err != nil {
		return err
	}
	sc, err := cl.NewStorageController[T](
		manager.shardID,
		sm,
		cl.NewMemoryStorage[T](manager.workqueueMeterProvider, opts...),
	)
	if err != nil {
		return err
	}

	cl.SetStorage[T](manager.storageSet, sc)
	controller.SetStorage(manager.storageSet)
	currentLoop := cl.New[T](controller, sc, cl.WithLogger(manager.logger))
	manager.registerController(currentLoop, sc)

	return nil
}

func (a *Manager) registerController(ctrl controller, receiver cl.Receiver) {
	a.controllers = append(a.controllers, &ctrlWrapper{controller: ctrl, receiver: receiver})
}
