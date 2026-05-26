package namespaces

import (
	"context"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type NamespaceHandler struct {
	ns_cache client.Reader
}

func NewNamespaceHandler(ns client.Reader) *NamespaceHandler {
	return &NamespaceHandler{
		ns_cache: ns,
	}
}

func (h *NamespaceHandler) List(namespace string) (runtime.Object, error) {
	namespaces := &corev1.NamespaceList{}
	err := h.ns_cache.List(context.Background(), namespaces, &client.ListOptions{})
	if err != nil {
		return nil, err
	}
	return namespaces, nil
}

func (h *NamespaceHandler) Get(namespace, name string) (runtime.Object, error) {
	ns := &corev1.Namespace{}
	err := h.ns_cache.Get(context.Background(), client.ObjectKey{Namespace: namespace, Name: name}, ns)
	if err != nil {
		return nil, err
	}
	return ns, nil
}
