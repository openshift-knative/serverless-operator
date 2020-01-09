package apis

import (
	routev1 "github.com/openshift/api/route/v1"
)

func init() {
	AddToSchemes = append(AddToSchemes, routev1.SchemeBuilder.AddToScheme)
}
