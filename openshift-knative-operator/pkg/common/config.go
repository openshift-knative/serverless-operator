package common

import (
	"knative.dev/operator/pkg/apis/operator/base"
)

// Configure sets a value in the given ConfigMap under the given key.
func Configure(s *base.CommonSpec, cm, key, value string) {
	if s.Config == nil {
		s.Config = make(map[string]map[string]string, 1)
	}

	if s.Config[cm] == nil {
		s.Config[cm] = make(map[string]string, 1)
	}

	s.Config[cm][key] = value
}

// ConfigureIfUnset sets a value in the given ConfigMap under the given key if it's not
// already set.
func ConfigureIfUnset(s *base.CommonSpec, cm, key, value string) {
	if s.Config == nil {
		s.Config = make(map[string]map[string]string, 1)
	}

	if s.Config[cm] == nil {
		s.Config[cm] = make(map[string]string, 1)
	}

	if _, ok := s.Config[cm][key]; ok {
		// Already set, nothing to do here.
		return
	}
	s.Config[cm][key] = value
}

// ConfigureIfUnsetDomain sets a value in the config-domain if it's not already set.
// config-domain can take a domain as a key so it is different from ConfigureIfUnset.
func ConfigureIfUnsetDefaultDomain(s *base.CommonSpec, key, value string) {
	const cm = "domain"
	if s.Config == nil {
		s.Config = make(map[string]map[string]string, 1)
	}

	if s.Config[cm] == nil {
		s.Config[cm] = make(map[string]string, 1)
	}

	if len(s.Config[cm]) != 0 {
		// Already set, nothing to do here.
		return
	}

	s.Config[cm][key] = value
}
