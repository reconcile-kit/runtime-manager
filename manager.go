package runtimemanager

import (
	"context"
	"fmt"
	"github.com/reconcile-kit/api/resource"
	cl "github.com/reconcile-kit/controlloop"
	event "github.com/reconcile-kit/redis-informer-provider"
	state "github.com/reconcile-kit/state-manager-provider"
	"net/http"
	"sync"
)

type InitReconciler[T resource.Object[T]] interface {
	cl.Reconcile[T]
	SetStorage(storage *cl.StorageSet)
}

type Stopped interface {
	Stop()
}

type Manager struct {
	shardID                string
	receivers              []cl.Receiver
	informerAddress        string
	informerAuthConfig     *InformerAuthConfig
	externalStorageAddress string
	storageSet             *cl.StorageSet
	stopped                []Stopped
	logger                 cl.Logger
	httpClient             *http.Client
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

func (a *Manager) addReceiver(receiver cl.Receiver) {
	a.receivers = append(a.receivers, receiver)
}
func (a *Manager) addStopped(stopped Stopped) {
	a.stopped = append(a.stopped, stopped)
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

func (a *Manager) Run(ctx context.Context) error {
	redisConfig := &event.RedisConfig{
		Addr: a.informerAddress,
	}
	if a.informerAuthConfig != nil {
		redisConfig.Username = a.informerAuthConfig.Username
		redisConfig.Password = a.informerAuthConfig.Password
		redisConfig.EnableTLS = a.informerAuthConfig.EnableTLS
	}
	eventProvider, err := event.NewRedisStreamListenerWithConfig(redisConfig, a.shardID)
	if err != nil {
		return err
	}

	informer := cl.NewStorageInformer(a.shardID, eventProvider, a.receivers)
	err = informer.Run(ctx)
	if err != nil {
		return err
	}
	return nil
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
		cl.NewMemoryStorage[T](opts...),
	)
	if err != nil {
		return err
	}

	cl.SetStorage[T](manager.storageSet, sc)
	controller.SetStorage(manager.storageSet)
	currentLoop := cl.New[T](controller, sc, cl.WithLogger(manager.logger))
	currentLoop.Run()
	manager.addReceiver(sc)
	manager.addStopped(currentLoop)

	return nil
}
