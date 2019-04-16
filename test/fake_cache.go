package test

import (
	"context"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type FakeCache struct {
	Err error
}

func (c *FakeCache) GetInformer(obj runtime.Object) (cache.Informer, error) {
	return nil, nil
}

func (c *FakeCache) GetInformerForKind(gvk schema.GroupVersionKind) (cache.Informer, error) {
	return nil, nil
}

func (c *FakeCache) Start(stopCh <-chan struct{}) error {
	return nil
}

func (c *FakeCache) WaitForCacheSync(stop <-chan struct{}) bool {
	return true
}

func (c *FakeCache) IndexField(obj runtime.Object, field string, extractValue client.IndexerFunc) error {
	return nil
}

func (c *FakeCache) Get(ctx context.Context, key client.ObjectKey, obj runtime.Object) error {
	return c.Err
}

func (c *FakeCache) List(ctx context.Context, list runtime.Object, opts ...client.ListOptionFunc) error {
	return nil
}
