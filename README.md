# Basic Kubernetes Pod Autoscaler

The default Horizontal Pod Autoscaler has several shortcomings which can be listed as follows:

1. The scaling is not restricted in terms of how machines are started and stopped at the same time.
2. The scaling action is not based on historical data but on current usage.
3. The thresholds for scaling up and scaling down is the same.

## Usage

Create a scaler object in the following format:

```yaml
apiVersion: arjunnaik.in/v1alpha1
kind: Scaler
metadata:
  name: example-scaler
  namespace: default
spec:
  evaluations: 2    // Number of evaluations in before scaling happens
  minReplicas: 1    // Minimum number of replicas
  maxReplicas: 10   // Maximum number of replicas
  scaleUp: 50       // Scale Up threshold in utilization percentage
  scaleDown: 20     // Scale Down threshold in utilization percentage
  scaleUpSize: 2    // Number of pods to scale up
  scaleDownSize: 1  // Number of pods to scale down
  target:
    kind: Deployment
    name: nginx
    apiVersion: apps/v1
```

In the above example the `target` field contains the scaling target. In this case the target is a _Deployment_ with 
the name `nginx`. Evaluations indicates the number of minutes _(cycles)_ before scaling happens. In this example,
if the CPU utilization of a pod is more than _50%_ for more than 2 minutes then the deployment is scaled up. The 
`scaleUpSize` and `scaleDownSize` indicates the number of pods to be increased on successful scale up or scale down
evaluations.

## Dependencies

This setup expects Prometheus to be running in the cluster and configured to scrape pod resource metrics. The address
for Prometheus can be passed through `-prometheus-url` flag.
