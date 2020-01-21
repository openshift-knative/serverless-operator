package common

import (
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
)

var Log = logf.Log.WithName("knative").WithName("openshift")

const MutationTimestampKey = "knative-serving-openshift/mutation"
