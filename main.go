package main

import (
	"flag"
	"github.com/arjunrn/dumb-scaler/controller"
	clientset "github.com/arjunrn/dumb-scaler/pkg/client/clientset/versioned"
	scalerinformers "github.com/arjunrn/dumb-scaler/pkg/client/informers/externalversions"
	"github.com/arjunrn/dumb-scaler/pkg/signals"
	"github.com/golang/glog"
	prometheus_api "github.com/prometheus/client_golang/api"
	log "github.com/sirupsen/logrus"
	cacheddiscovery "k8s.io/client-go/discovery/cached"
	"k8s.io/client-go/dynamic"
	kubeinformers "k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/restmapper"
	"k8s.io/client-go/scale"
	"k8s.io/client-go/tools/clientcmd"
	"time"
)

var (
	masterURL     string
	kubeconfig    string
	prometheusURL string
	resyncInterval int
)


func main() {
	flag.Parse()

	logger := log.New()
	glog.SetLogger(logger.WithField("foo", "bar"))

	log.Info("starting the main()")
	stopCh := signals.SetupSignalHandler()

	cfg, err := clientcmd.BuildConfigFromFlags(masterURL, kubeconfig)
	if err != nil {
		log.Fatalf("Error building kubeconfig: %s", err.Error())
	}

	kubeClient, err := kubernetes.NewForConfig(cfg)

	if err != nil {
		log.Fatalf("Error building kubernetes clientset: %s", err.Error())
	}

	scalerClient, err := clientset.NewForConfig(cfg)
	if err != nil {
		log.Fatalf("error building scaler clientset: %s", err.Error())
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

	if err != nil {
		log.Fatalf("Failed to create scale getter: %s", err.Error())
	}

	podInformer := kubeInformerFactory.Core().V1().Pods()

	config := prometheus_api.Config{Address: prometheusURL}
	prometheusClient, err := prometheus_api.NewClient(config)
	if err != nil {
		log.Fatalf("failed to create prometheus client with address: %s", prometheusURL)
	}

	interval:=time.Duration(resyncInterval)*time.Second

	controller := controller.NewController(kubeClient, scalerClient, scalerInformerFactory.Arjunnaik().V1alpha1().Scalers(),
		podInformer, scaleGetter, mapper, prometheusClient, interval)

	go kubeInformerFactory.Start(stopCh)
	go scalerInformerFactory.Start(stopCh)

	if err = controller.Run(2, stopCh); err != nil {
		log.Fatalf("error running scaler controller: %v", err.Error())
	}
}

func init() {
	flag.StringVar(&kubeconfig, "kubeconfig", "", "Path to a kubeconfig. Only required if out-of-cluster.")
	flag.StringVar(&masterURL, "master", "", "The address of the Kubernetes API server. Overrides any value in kubeconfig. Only required if out-of-cluster.")
	flag.StringVar(&prometheusURL, "prometheus-url", "", "Address of the prometheus server")
	flag.IntVar(&resyncInterval, "resync-interval",30,"The resync interval for the controller in seconds")
}
