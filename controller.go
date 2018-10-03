package main

import (
	"fmt"
	clientset "github.com/arjunrn/dumb-scaler/pkg/client/clientset/versioned"
	informers "github.com/arjunrn/dumb-scaler/pkg/client/informers/externalversions/scaler/v1alpha1"
	"github.com/golang/glog"
	"k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
	"time"
)

const controllerAgentName = "scaler-controller"

// Controller is the controller implementation for Foo resources
type Controller struct {
	// kubeclientset is a standard kubernetes clientset
	kubeclientset kubernetes.Interface
	// scalerclientset is a clientset for our own API group
	scalerclientset clientset.Interface

	workQueue     workqueue.RateLimitingInterface
	scalersSynced cache.InformerSynced
}

// NewController returns a new sample controller
func NewController(kubeclientset kubernetes.Interface, sampleclientset clientset.Interface, scalerInfomer informers.ScalerInformer) *Controller {
	controller := &Controller{
		kubeclientset:   kubeclientset,
		scalerclientset: sampleclientset,
		workQueue:       workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "Scalers"),

		scalersSynced: scalerInfomer.Informer().HasSynced,
	}

	glog.Info("Setting up event handlers")
	scalerInfomer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:controller.enqueueScaler,
		UpdateFunc: func(oldObj, newObj interface{}) {
			controller.enqueueScaler(newObj)
		},
	})
	return controller
}

// Run will set up the event handlers for types we are interested in, as well
// as syncing informer caches and starting workers. It will block until stopCh
// is closed, at which point it will shutdown the workqueue and wait for
// workers to finish processing their current work items.
func (c *Controller) Run(threadiness int, stopCh <-chan struct{}) error {
	defer runtime.HandleCrash()
	defer c.workQueue.ShutDown()

	// Start the informer factories to begin populating the informer caches
	glog.Info("Starting Scaler controller")

	glog.Info("Waiting for informer caches to be synced")
	if ok := cache.WaitForCacheSync(stopCh, c.scalersSynced); !ok {
		return fmt.Errorf("failed to wait for caches to sync")
	}

	glog.Info("starting workers")
	for i := 0; i < threadiness; i++ {
		go wait.Until(c.runWorker, time.Second, stopCh)
	}

	glog.Info("Started workers")
	<-stopCh
	glog.Info("Shutting down workers")

	return nil
}

func (c *Controller) runWorker() {
	for c.processNextWorkItem() {

	}
}

func (c *Controller) processNextWorkItem() bool {
	obj, shutdown := c.workQueue.Get()
	if shutdown {
		return false
	}
	err := func(obj interface{}) error {
		defer c.workQueue.Done(obj)

		if key, ok := obj.(string); ok {
			glog.Infof("processing %s", key)
			c.workQueue.Forget(obj)
			glog.Infof("forgetting %v", obj)
		}
		glog.Info("finished processing")
		return nil
	}(obj)
	if err != nil {
		runtime.HandleError(err)
		return true
	}
	return true
}

func (c *Controller) enqueueScaler(obj interface{}) {
	var key string
	var err error
	if key, err = cache.MetaNamespaceKeyFunc(obj); err != nil {
		runtime.HandleError(err)
		return
	}
	c.workQueue.AddRateLimited(key)
}
