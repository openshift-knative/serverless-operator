# OpenShift Serverless Knative Serving Control Plane Readiness

## Severity: Critical

## Impact

Any users or services using Knative Serving will be unable to get their Knative Services reconciled.
This means, any new Knative Services might not be created, and any existing Knative Services might not be updated. Also, it might also mean that the user workload is not being served.

## Summary

OpenShift Serverless does not create any alerts out-of-the box. However, users can create alerts based on the metrics exposed.

The metric `knative_up{namespace="openshift-serverless", type="serving_status"}` should always have a value of `1`. 

This failure should only occur in extremely catastrophic scenarios as all components are deployed in HA configuration.

Possible causes can be:
- Network issues in the cluster
- Resource starvation on the cluster
- OpenShift Serverless upgrade issues

You should check the control plane status of OpenShift Serverless operator, Knative Serving and the used Knative Serving Ingress.

## Prerequisites

1. You must have admin access to the cluster via `oc` CLI.

2. You must find out which Knative Serving Ingress is used.

Run the following command to see the Knative Serving spec's ingress related section:

```shell
$ k get knativeserving -n knative-serving knative-serving -o jsonpath='{.spec.ingress}'
```

If the output is empty, then Kourier is used as the Knative Serving Ingress. 

If the output is not empty, check which Ingress is used by looking at which ingress has `enabled: true`:

```yaml
  contour:
    enabled: false
  istio:
    enabled: true
  kourier:
    enabled: false
```

In this case, Istio is used as the Knative Serving Ingress.

## Steps

1. Check to see if all the pods are running in `openshift-serverless` namespace:

```bash
$ oc -n openshift-serverless get pods
```

2. If they are not running, look at the pod's logs/events to see what may be causing the issues. Please make sure to grab the logs/events so they can be shared with the engineering team later:

```bash
# Get Pod names as we don't have a common label
$ SERVERLESS_OPERATOR_POD_NAMES_STR=$(oc -n openshift-serverless get pods --template '{{range .items}}{{.metadata.name}}{{" "}}{{end}}')
$ SERVERLESS_OPERATOR_POD_NAMES=($(echo "$SERVERLESS_OPERATOR_POD_NAMES_STR" | tr ' ' '\n'))

# Check pod logs
$ for element in "${SERVERLESS_OPERATOR_POD_NAMES[@]}"
$ do
$     oc -n openshift-serverless logs "$element" --prefix=true
$ done


# Check events 
$ oc -n openshift-serverless get events

# Check pod status fields
$ oc -n openshift-serverless get pods -o jsonpath="{range .items[*]}{.status}{\"\n\n\"}{end}" 
```

3. Redeploy any failing controllers by restarting the deployments:

```bash
$ oc -n openshift-serverless rollout restart deployments/knative-openshift
$ oc -n openshift-serverless rollout restart deployments/knative-openshift-ingress
$ oc -n openshift-serverless rollout restart deployments/knative-operator-webhook
```

This should result in new pods getting deployed, attempt step (1) again and see if the pods achieve running state.

5. Check to see if all the pods are running in `knative-serving` namespace:

```bash
$ oc -n knative-serving get pods -l app.kubernetes.io/name=knative-serving
``` 

6. If they are not running, look at the pod's logs/events to see what may be causing the issues. Please make sure to grab the logs/events so they can be shared with the engineering team later:

```bash
# Check pod logs 
$ oc -n knative-serving logs -l app.kubernetes.io/name=knative-serving --prefix=true

# Check events 
$ oc -n knative-serving get events | grep pod

# Check pod status fields
$ oc -n knative-serving get pods -l app.kubernetes.io/name=knative-serving -o jsonpath="{range .items[*]}{.status}{\"\n\n\"}{end}"
```

7. Redeploy Knative Serving controllers by restarting the deployments:

```bash
$ oc -n knative-serving rollout restart deployments -l app.kubernetes.io/name=knative-serving
```

This should result in new pods getting deployed, attempt step (5) again and see if the pods achieve running state.

8. If Kourier is used as the Knative Serving Ingress, check to see if all the relevant pods are running in `knative-serving-ingress` namespace:

```bash
$ oc -n knative-serving-ingress get pods -l 'app in (3scale-kourier-gateway, net-kourier-controller)'
```

9. If they are not running, look at the pod's logs/events to see what may be causing the issues. Please make sure to grab the logs/events so they can be shared with the engineering team later:

```bash
# Check pod logs 
$ oc -n knative-serving-ingress logs -l 'app in (3scale-kourier-gateway, net-kourier-controller)' --prefix=true

# Check events 
$ oc -n knative-serving-ingress get events | grep pod

# Check pod status fields
$ oc -n knative-serving-ingress get pods -l 'app in (3scale-kourier-gateway, net-kourier-controller)' -o jsonpath="{range .items[*]}{.status}{\"\n\n\"}{end}"
```

10. Redeploy Knative Serving Ingress controllers by restarting the deployments:

```bash
$ oc -n knative-serving-ingress rollout restart deployments -l app.kubernetes.io/component=net-kourier
```

This should result in new pods getting deployed, attempt step (8) again and see if the pods achieve running state.

11. If Istio is used as the Knative Serving Ingress, check Istio status. You may find the SOPs for Istio by contacting Istio support. 

12. If the problem persists, capture the logs and escalate to OpenShift Serverless engineering team with a Knative ["must-gather"](https://github.com/openshift-knative/must-gather) dump.


