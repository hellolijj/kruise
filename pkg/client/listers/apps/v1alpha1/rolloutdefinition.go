/*
Copyright The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

// Code generated by lister-gen. DO NOT EDIT.

package v1alpha1

import (
	v1alpha1 "github.com/hellolijj/kruise/pkg/apis/apps/v1alpha1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/tools/cache"
)

// RolloutDefinitionLister helps list RolloutDefinitions.
type RolloutDefinitionLister interface {
	// List lists all RolloutDefinitions in the indexer.
	List(selector labels.Selector) (ret []*v1alpha1.RolloutDefinition, err error)
	// RolloutDefinitions returns an object that can list and get RolloutDefinitions.
	RolloutDefinitions(namespace string) RolloutDefinitionNamespaceLister
	RolloutDefinitionListerExpansion
}

// rolloutDefinitionLister implements the RolloutDefinitionLister interface.
type rolloutDefinitionLister struct {
	indexer cache.Indexer
}

// NewRolloutDefinitionLister returns a new RolloutDefinitionLister.
func NewRolloutDefinitionLister(indexer cache.Indexer) RolloutDefinitionLister {
	return &rolloutDefinitionLister{indexer: indexer}
}

// List lists all RolloutDefinitions in the indexer.
func (s *rolloutDefinitionLister) List(selector labels.Selector) (ret []*v1alpha1.RolloutDefinition, err error) {
	err = cache.ListAll(s.indexer, selector, func(m interface{}) {
		ret = append(ret, m.(*v1alpha1.RolloutDefinition))
	})
	return ret, err
}

// RolloutDefinitions returns an object that can list and get RolloutDefinitions.
func (s *rolloutDefinitionLister) RolloutDefinitions(namespace string) RolloutDefinitionNamespaceLister {
	return rolloutDefinitionNamespaceLister{indexer: s.indexer, namespace: namespace}
}

// RolloutDefinitionNamespaceLister helps list and get RolloutDefinitions.
type RolloutDefinitionNamespaceLister interface {
	// List lists all RolloutDefinitions in the indexer for a given namespace.
	List(selector labels.Selector) (ret []*v1alpha1.RolloutDefinition, err error)
	// Get retrieves the RolloutDefinition from the indexer for a given namespace and name.
	Get(name string) (*v1alpha1.RolloutDefinition, error)
	RolloutDefinitionNamespaceListerExpansion
}

// rolloutDefinitionNamespaceLister implements the RolloutDefinitionNamespaceLister
// interface.
type rolloutDefinitionNamespaceLister struct {
	indexer   cache.Indexer
	namespace string
}

// List lists all RolloutDefinitions in the indexer for a given namespace.
func (s rolloutDefinitionNamespaceLister) List(selector labels.Selector) (ret []*v1alpha1.RolloutDefinition, err error) {
	err = cache.ListAllByNamespace(s.indexer, s.namespace, selector, func(m interface{}) {
		ret = append(ret, m.(*v1alpha1.RolloutDefinition))
	})
	return ret, err
}

// Get retrieves the RolloutDefinition from the indexer for a given namespace and name.
func (s rolloutDefinitionNamespaceLister) Get(name string) (*v1alpha1.RolloutDefinition, error) {
	obj, exists, err := s.indexer.GetByKey(s.namespace + "/" + name)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, errors.NewNotFound(v1alpha1.Resource("rolloutdefinition"), name)
	}
	return obj.(*v1alpha1.RolloutDefinition), nil
}
