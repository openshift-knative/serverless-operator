package consoleserver

import (
	yaml2 "github.com/ghodss/yaml"
	"github.com/openshift/library-go/pkg/operator/resource/resourcemerge"
)

type ConsoleYAMLMerger struct{}

func (b *ConsoleYAMLMerger) Merge(configYAMLs ...[]byte) (converted []byte, conversionErr error) {
	mergedConfig, err := b.combine(configYAMLs...)
	if err != nil {
		return nil, err
	}
	return yaml2.JSONToYAML(mergedConfig)
}

func (b *ConsoleYAMLMerger) combine(configYAMLs ...[]byte) (mergedConfig []byte, mergeError error) {
	mergedConfig, err := resourcemerge.MergeProcessConfig(nil, configYAMLs...)
	if err != nil {
		return nil, err
	}
	return mergedConfig, nil
}
