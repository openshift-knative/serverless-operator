/*
Copyright 2021 The Knative Authors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

	http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package feature

const (
	KReferenceGroup          = "kreference-group"
	DeliveryRetryAfter       = "delivery-retryafter"
	DeliveryTimeout          = "delivery-timeout"
	KReferenceMapping        = "kreference-mapping"
	TransportEncryption      = "transport-encryption"
	EvenTypeAutoCreate       = "eventtype-auto-create"
	OIDCAuthentication       = "authentication-oidc"
	NodeSelectorLabel        = "apiserversources-nodeselector-"
	CrossNamespaceEventLinks = "cross-namespace-event-links"
	NewAPIServerFilters      = "new-apiserversource-filters"
	AuthorizationDefaultMode = "default-authorization-mode"
	OIDCDiscoveryBaseURL     = "oidc-discovery-base-url"
)
