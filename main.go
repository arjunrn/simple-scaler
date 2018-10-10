package main

import (
	"flag"
	"github.com/arjunrn/dumb-scaler/controller"
	clientset "github.com/arjunrn/dumb-scaler/pkg/client/clientset/versioned"
	scalerinformers "github.com/arjunrn/dumb-scaler/pkg/client/informers/externalversions"
	"github.com/arjunrn/dumb-scaler/pkg/signals"
	"github.com/golang/glog"
	cacheddiscovery "k8s.io/client-go/discovery/cached"
	"k8s.io/client-go/dynamic"
	kubeinformers "k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/restmapper"
	"k8s.io/client-go/scale"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/kubernetes/pkg/controller/podautoscaler/metrics"
	resourceclient "k8s.io/metrics/pkg/client/clientset/versioned/typed/metrics/v1beta1"
	"k8s.io/metrics/pkg/client/custom_metrics"
	"k8s.io/metrics/pkg/client/external_metrics"
	"time"
)

var (
	masterURL  string
	kubeconfig string
)

func main() {
	flag.Parse()
	glog.Info("starting the main()")
	stopCh := signals.SetupSignalHandler()

	cfg, err := clientcmd.BuildConfigFromFlags(masterURL, kubeconfig)
	if err != nil {
		glog.Fatalf("Error building kubeconfig: %s", err.Error())
	}

	kubeClient, err := kubernetes.NewForConfig(cfg)

	if err != nil {
		glog.Fatalf("Error building kubernetes clientset: %s", err.Error())
	}

	scalerClient, err := clientset.NewForConfig(cfg)
	if err != nil {
		glog.Fatalf("error building scaler clientset: %s", err.Error())
	}

	kubeInformerFactory := kubeinformers.NewSharedInformerFactory(kubeClient, time.Second*30)
	scalerInformerFactory := scalerinformers.NewSharedInformerFactory(scalerClient, time.Second*30)



	cachedClient := cacheddiscovery.NewMemCacheClient(kubeClient.Discovery())
	// TODO: understand what this caching is all about and why its needed
	cachedClient.Invalidate()
	mapper := restmapper.NewDeferredDiscoveryRESTMapper(cachedClient)
	// TODO: figure out what this discovery shit is
	scaleKindResolver := scale.NewDiscoveryScaleKindResolver(kubeClient.Discovery())

	scaleGetter, err := scale.NewForConfig(cfg, mapper, dynamic.LegacyAPIPathResolverFunc, scaleKindResolver)

	apiVersionsGetter := custom_metrics.NewAvailableAPIsGetter(scalerClient.Discovery())
	metricsClient := metrics.NewRESTMetricsClient(
		resourceclient.NewForConfigOrDie(cfg),
		custom_metrics.NewForConfig(cfg, mapper, apiVersionsGetter),
		external_metrics.NewForConfigOrDie(cfg),
	)

	if err != nil {
		glog.Fatalf("Failed to create scale getter: %s", err.Error())
	}

	podInformer := kubeInformerFactory.Core().V1().Pods()

	controller := controller.NewController(kubeClient, scalerClient, scalerInformerFactory.Arjunnaik().V1alpha1().Scalers(),
		podInformer, metricsClient, scaleGetter, mapper)

	go kubeInformerFactory.Start(stopCh)
	go scalerInformerFactory.Start(stopCh)

	if err = controller.Run(2, stopCh); err != nil {
		glog.Fatalf("error running scaler controller: %v", err.Error())
	}
}

func init() {
	flag.StringVar(&kubeconfig, "kubeconfig", "", "Path to a kubeconfig. Only required if out-of-cluster.")
	flag.StringVar(&masterURL, "master", "", "The address of the Kubernetes API server. Overrides any value in kubeconfig. Only required if out-of-cluster.")
}
