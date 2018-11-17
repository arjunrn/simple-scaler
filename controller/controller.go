package controller

import (
	"fmt"
	"github.com/arjunrn/dumb-scaler/pkg/apis/scaler/v1alpha1"
	clientset "github.com/arjunrn/dumb-scaler/pkg/client/clientset/versioned"
	informers "github.com/arjunrn/dumb-scaler/pkg/client/informers/externalversions/scaler/v1alpha1"
	"github.com/arjunrn/dumb-scaler/pkg/replicacalculator"
	"github.com/golang/glog"
	autoscalingv1 "k8s.io/api/autoscaling/v1"
	autoscalingv2 "k8s.io/api/autoscaling/v2beta2"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	apimeta "k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	coreinformers "k8s.io/client-go/informers/core/v1"
	"k8s.io/client-go/kubernetes"
	scaleclient "k8s.io/client-go/scale"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
	"k8s.io/kubernetes/pkg/controller/podautoscaler/metrics"
	"time"
)

// Controller is the controller implementation for Foo resources
type Controller struct {
	// kubeclientset is a standard kubernetes clientset
	kubeclientset kubernetes.Interface
	// scalerclientset is a clientset for our own API group
	scalerclientset clientset.Interface

	queue           workqueue.RateLimitingInterface
	scalersSynced   cache.InformerSynced
	metricsclient   metrics.MetricsClient
	mapper          apimeta.RESTMapper
	scaleNamespacer scaleclient.ScalesGetter
	replicaCalc     *replicacalculator.ReplicaCalculator
	deploymentCache *replicacalculator.DeploymentCache
}

// NewController returns a new sample controller
func NewController(kubeclientset kubernetes.Interface, scalerclientset clientset.Interface,
	scalerInformer informers.ScalerInformer, podInformer coreinformers.PodInformer, metricsclient metrics.MetricsClient,
	scaleNamespacer scaleclient.ScalesGetter, mapper apimeta.RESTMapper, resyncInterval time.Duration) *Controller {
	controller := &Controller{
		kubeclientset:   kubeclientset,
		scalerclientset: scalerclientset,
		queue:           workqueue.NewNamedRateLimitingQueue(NewDefaultScalerRateLimiter(resyncInterval), "scalers"),
		scalersSynced:   scalerInformer.Informer().HasSynced,
		metricsclient:   metricsclient,
		scaleNamespacer: scaleNamespacer,
	}
	controller.deploymentCache = replicacalculator.NewDeploymentCache(15, 15*time.Minute)
	controller.mapper = mapper
	podLister := podInformer.Lister()
	controller.replicaCalc = replicacalculator.NewReplicaCalculator(metricsclient, podLister, controller.deploymentCache)
	glog.Info("Setting up event handlers")
	scalerInformer.Informer().AddEventHandlerWithResyncPeriod(cache.ResourceEventHandlerFuncs{
		AddFunc: controller.enqueueScaler,
		UpdateFunc: func(oldObj, newObj interface{}) {
			controller.enqueueScaler(newObj)
		},
	}, resyncInterval)

	return controller
}

// Run will set up the event handlers for types we are interested in, as well
// as syncing informer caches and starting workers. It will block until stopCh
// is closed, at which point it will shutdown the workqueue and wait for
// workers to finish processing their current work items.
func (c *Controller) Run(threadiness int, stopCh <-chan struct{}) error {
	defer runtime.HandleCrash()
	defer c.queue.ShutDown()

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
	key, shutdown := c.queue.Get()
	if shutdown {
		return false
	}
	defer c.queue.Done(key)

	err := c.reconcileKey(key.(string))
	if err == nil {
		// don't "forget" here because we want to only process a given HPA once per resync interval
		return true
	}

	c.queue.AddRateLimited(key)
	utilruntime.HandleError(err)
	return true
}

func (c *Controller) reconcileKey(key string) error {
	namespace, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		return err
	}

	scaler, err := c.scalerclientset.ArjunnaikV1alpha1().Scalers(namespace).Get(name, metav1.GetOptions{})
	if errors.IsNotFound(err) {
		glog.Errorf("Scaler %s has been deleted", name)
		return nil
	}
	return c.reconcileScaler(scaler)
}

func (c *Controller) enqueueScaler(obj interface{}) {
	var key string
	var err error
	if key, err = cache.MetaNamespaceKeyFunc(obj); err != nil {
		runtime.HandleError(err)
		return
	}
	c.queue.AddRateLimited(key)
}

func (c *Controller) reconcileScaler(scalerShared *v1alpha1.Scaler) error {
	scaler := scalerShared.DeepCopy()
	version, err := schema.ParseGroupVersion(scaler.Spec.Target.APIVersion)
	if err != nil {
		return err
	}
	targetGK := schema.GroupKind{
		Group: version.Group,
		Kind:  scaler.Spec.Target.Kind,
	}
	mappings, err := c.mapper.RESTMappings(targetGK)
	if err != nil {
		return err
	}
	glog.Infof("Found mappings: %v", mappings)
	scale, targetGR, err := c.scaleForResourceMappings(scaler.Namespace, scaler.Spec.Target.Name, mappings)
	if err != nil {
		return err
	}
	glog.Infof("Found scale: %v target group: %v", scale.Name, targetGR.Resource)

	currentReplicas := scale.Status.Replicas
	desiredReplicas := int32(0)
	rescale := true
	if scale.Spec.Replicas == 0 {
		glog.Infof("autoscaling disabled by target. %v", scale)
		rescale = false
	} else if currentReplicas > scaler.Spec.MaxReplicas {
		desiredReplicas = scaler.Spec.MaxReplicas
	} else if currentReplicas < scaler.Spec.MinReplicas {
		desiredReplicas = scaler.Spec.MinReplicas
	} else if currentReplicas == 0 {
		desiredReplicas = 1
	} else {
		replicas, _, _, _, err := c.computeReplicasForMetrics(scaler, scale, scaler.Spec.ScaleUp, scaler.Spec.ScaleDown)
		if err == nil {
			desiredReplicas = replicas
		} else {
			glog.Errorf("error computing replicas: %v", err)
		}

	}
	glog.Infof("currentReplicas: %d desiredReplicas: %d, rescale: %v", currentReplicas, desiredReplicas, rescale)

	if desiredReplicas < scaler.Spec.MinReplicas {
		glog.Infof("cannot scaled down more than min replicas")
		return nil
	}

	if desiredReplicas > scaler.Spec.MaxReplicas {
		glog.Infof("cannot scale up more than max replicas")
		return nil
	}

	scale.Spec.Replicas = desiredReplicas
	_, err = c.scaleNamespacer.Scales(scale.Namespace).Update(targetGR, scale)
	if err != nil {
		return err
	}
	c.deploymentCache.AddEvent(scaler.Name, currentReplicas, desiredReplicas)

	scaler.Status.Condition = fmt.Sprintf("Scaled to %d replicas", desiredReplicas)
	_, err = c.scalerclientset.ArjunnaikV1alpha1().Scalers(scaler.Namespace).Update(scaler)
	if err != nil {
		glog.Errorf("Failed to Update Scaler Status %v", err)
	}
	return nil
}

func (c *Controller) scaleForResourceMappings(namespace, name string, mappings []*apimeta.RESTMapping) (*autoscalingv1.Scale, schema.GroupResource, error) {
	var firstErr error
	for i, mapping := range mappings {
		targetGR := mapping.Resource.GroupResource()
		scale, err := c.scaleNamespacer.Scales(namespace).Get(targetGR, name)
		if err == nil {
			return scale, targetGR, nil
		}

		// if this is the first error, remember it,
		// then go on and try other mappings until we find a good one
		if i == 0 {
			firstErr = err
		}
	}

	// make sure we handle an empty set of mappings
	if firstErr == nil {
		firstErr = fmt.Errorf("unrecognized resource")
	}

	return nil, schema.GroupResource{}, firstErr

}
func (c *Controller) computeReplicasForMetrics(scaler *v1alpha1.Scaler, scale *autoscalingv1.Scale, scaleUpCpu, scaleDownCpu int32,
) (replicas int32, metric string, status *autoscalingv2.MetricStatus, timestamp time.Time, err error) {
	currentReplicas := scale.Status.Replicas

	if scale.Status.Selector == "" {
		glog.Errorf("Target needs a selector: %v", scale)
		return 0, "", nil, time.Time{}, fmt.Errorf("selector required")
	}

	selector, err := labels.Parse(scale.Status.Selector)

	if err != nil {
		errMsg := fmt.Sprintf("couldn't convert selector into a corresponding internal selector object: %v", err)
		return 0, "", nil, time.Time{}, fmt.Errorf(errMsg)
	}

	replicaCountProposal, timestampProposal, _, err := c.computeStatusForResourceMetric(currentReplicas, scaleUpCpu, scaleDownCpu, scaler, selector)
	if err != nil {
		return 0, "error", nil, timestampProposal, err
	}
	return replicaCountProposal, "test", nil, timestampProposal, nil
}
func (c *Controller) computeStatusForResourceMetric(currentReplicas int32, scaleUpCpu int32, scaleDownCpu int32, scaler *v1alpha1.Scaler, selector labels.Selector) (int32, time.Time, string, error) {
	// TODO: this is a hack. In the main controller the ResourceName is part of the metricSpec. Why?
	name := corev1.ResourceName("cpu")
	replicaCountProposal, timestampProposal, err := c.replicaCalc.GetResourceReplicas(currentReplicas, scaleUpCpu, scaleDownCpu, name, scaler, selector)
	if err != nil {
		return 0, time.Time{}, "", err
	}
	return replicaCountProposal, timestampProposal, "", nil
}

func (c *Controller) updateStatus(scaler *v1alpha1.Scaler) error {

	return nil
}
