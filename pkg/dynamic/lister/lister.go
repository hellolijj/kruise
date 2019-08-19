package lister

import (
	"fmt"

	"k8s.io/klog"

	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/tools/cache"
)

type Lister struct {
	indexer       cache.Indexer
	groupResource schema.GroupResource
}

func New(groupResource schema.GroupResource, indexer cache.Indexer) *Lister {
	return &Lister{
		groupResource: groupResource,
		indexer:       indexer,
	}
}

func (l *Lister) List(selector labels.Selector) (ret []*unstructured.Unstructured, err error) {
	err = cache.ListAll(l.indexer, selector, func(obj interface{}) {
		ret = append(ret, obj.(*unstructured.Unstructured))
	})
	return ret, err
}

func (l *Lister) ListNamespace(namespace string, selector labels.Selector) (ret []*unstructured.Unstructured, err error) {
	err = cache.ListAllByNamespace(l.indexer, namespace, selector, func(obj interface{}) {
		ret = append(ret, obj.(*unstructured.Unstructured))
	})
	return ret, err
}

func (l *Lister) Get(namespace, name string) (*unstructured.Unstructured, error) {
	key := name
	if namespace != "" {
		key = fmt.Sprintf("%s/%s", namespace, name)
	}
	v := l.indexer.ListKeys()
	klog.Infof("qwkLog: indexer.listkeys = %v", v)
	obj, exists, err := l.indexer.GetByKey(key)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, errors.NewNotFound(l.groupResource, key)
	}
	return obj.(*unstructured.Unstructured), nil
}
