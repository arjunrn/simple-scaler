package main

import (
	"flag"
	clientset "github.com/arjunrn/dumb-scaler/pkg/client/clientset/versioned"
	scalerinformers "github.com/arjunrn/dumb-scaler/pkg/client/informers/externalversions"
	"github.com/arjunrn/dumb-scaler/pkg/signals"
	"github.com/golang/glog"
	kubeinformers "k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
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

	controller := NewController(kubeClient, scalerClient, scalerInformerFactory.Arjunnaik().V1alpha1().Scalers())

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
