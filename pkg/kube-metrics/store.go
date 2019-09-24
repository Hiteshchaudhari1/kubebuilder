package kubemetrics

import (
	"context"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/tools/cache"
	"k8s.io/kube-state-metrics/pkg/metric"
	metricsstore "k8s.io/kube-state-metrics/pkg/metrics_store"
)

// NewMetricsStores returns collections of metrics in the namespaces provided, per the api/kind resource.
// The metrics are registered in the custom generateStore function that needs to be defined.
func NewMetricsStores(dclient dynamic.NamespaceableResourceInterface, namespaces []string, api string, kind string, metricFamily []metric.FamilyGenerator) []*metricsstore.MetricsStore {
	namespaces = deduplicateNamespaces(namespaces)
	var stores []*metricsstore.MetricsStore
	// Generate collector per namespace.
	for _, ns := range namespaces {
		composedMetricGenFuncs := metric.ComposeMetricGenFuncs(metricFamily)
		headers := metric.ExtractMetricFamilyHeaders(metricFamily)
		store := metricsstore.NewMetricsStore(headers, composedMetricGenFuncs)
		reflectorPerNamespace(context.TODO(), dclient, &unstructured.Unstructured{}, store, ns)
		stores = append(stores, store)
	}
	return stores
}

func deduplicateNamespaces(ns []string) (list []string) {
	keys := make(map[string]struct{})
	for _, entry := range ns {
		if _, ok := keys[entry]; !ok {
			keys[entry] = struct{}{}
			list = append(list, entry)
		}
	}
	return list
}

func reflectorPerNamespace(
	ctx context.Context,
	dynamicInterface dynamic.NamespaceableResourceInterface,
	expectedType interface{},
	store cache.Store,
	ns string,
) {
	lw := listWatchFunc(dynamicInterface, ns)
	reflector := cache.NewReflector(&lw, expectedType, store, 0)
	go reflector.Run(ctx.Done())
}

func listWatchFunc(dynamicInterface dynamic.NamespaceableResourceInterface, namespace string) cache.ListWatch {
	return cache.ListWatch{
		ListFunc: func(opts metav1.ListOptions) (runtime.Object, error) {
			return dynamicInterface.Namespace(namespace).List(opts)
		},
		WatchFunc: func(opts metav1.ListOptions) (watch.Interface, error) {
			return dynamicInterface.Namespace(namespace).Watch(opts)
		},
	}
}
