package kube

import (
	"os"
	"path/filepath"

	"time"

	dapisv1 "github.com/softcdata/testudo-operator/pkg/apis/disaster/v1"
	"github.com/softcdata/testudo-operator/pkg/clientset/versioned"
	"github.com/softcdata/testudo-operator/pkg/informers/externalversions"
	velerov1 "github.com/vmware-tanzu/velero/pkg/apis/velero/v1"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"sigs.k8s.io/controller-runtime/pkg/cluster"
)

type KubeClient struct {
	K8sClient       kubernetes.Interface
	DisasterClient  versioned.Interface
	DynamicClient   dynamic.Interface
	ClusterClient   cluster.Cluster
	Config          *rest.Config
	InformerFactory externalversions.SharedInformerFactory
}

func NewClient() (KubeClient, error) {

	var kc KubeClient
	config, err := rest.InClusterConfig()
	if err != nil {
		kubeconfig := filepath.Join(homeDir(), ".kube", "config")

		config, err = clientcmd.BuildConfigFromFlags("", kubeconfig)
		if err != nil {
			return kc, err
		}
	}

	// Increase QPS and Burst to avoid client-side throttling
	config.QPS = 100
	config.Burst = 200
	kc.Config = rest.CopyConfig(config)

	disasterClient, err := versioned.NewForConfig(config)
	if err != nil {
		return kc, err
	}
	kc.DisasterClient = disasterClient

	k8sClient, err := kubernetes.NewForConfig(config)
	if err != nil {
		return kc, err
	}
	kc.K8sClient = k8sClient

	kc.InformerFactory = externalversions.NewSharedInformerFactory(disasterClient, time.Minute*10)

	dynamicClient, err := dynamic.NewForConfig(config)
	if err != nil {
		return kc, err
	}
	kc.DynamicClient = dynamicClient

	// Create Scheme with all required types
	scheme := runtime.NewScheme()
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(dapisv1.AddToScheme(scheme))
	utilruntime.Must(velerov1.AddToScheme(scheme))

	cl, err := cluster.New(config, func(o *cluster.Options) {
		o.Scheme = scheme
	})
	if err != nil {
		return kc, err
	}

	kc.ClusterClient = cl

	return kc, nil
}

func homeDir() string {
	if h := os.Getenv("HOME"); h != "" {
		return h
	}
	return os.Getenv("USERPROFILE") // Windows
}
