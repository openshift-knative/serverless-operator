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

You should check both OpenShift Serverless operator and Knative Serving control plane status.

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

8. If the problem persists, capture the logs and escalate to OpenShift Serverless engineering team.


