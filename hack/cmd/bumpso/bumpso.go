package main

import (
	"bytes"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/coreos/go-semver/semver"
	"gopkg.in/yaml.v3"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	project := make(map[string]interface{}, 8)
	wd, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}
	projectPath := filepath.Join(wd, "olm-catalog/serverless-operator/project.yaml")

	flag.StringVar(&projectPath, "project-path", projectPath, "")
	flag.Parse()

	var node yaml.Node

	file, err := os.ReadFile(projectPath)
	if err != nil {
		return fmt.Errorf("failed to read file %s: %w", projectPath, err)
	}

	if err := yaml.NewDecoder(bytes.NewBuffer(file)).Decode(&project); err != nil {
		return fmt.Errorf("failed to decode file into map: %w", err)
	}
	if err := yaml.NewDecoder(bytes.NewBuffer(file)).Decode(&node); err != nil {
		return fmt.Errorf("failed to decode file into node: %w", err)
	}

	previousVersion, err := previousVersion(project)
	if err != nil {
		return err
	}

	currentVersion, err := currentVersion(project)
	if err != nil {
		return err
	}
	// We don't care about path versions (patch versions are potentially skipped, etc)
	currentVersion.Patch = 0

	newVersion := &semver.Version{
		Major:      currentVersion.Major,
		Minor:      currentVersion.Minor,
		Patch:      currentVersion.Patch,
		PreRelease: currentVersion.PreRelease,
		Metadata:   currentVersion.Metadata,
	}
	newVersion.BumpMinor()

	upgradeSequence, _, err := unstructured.NestedSlice(project, "upgrade_sequence")
	if err != nil {
		return err
	}
	channelsList, _, err := unstructured.NestedSlice(project, "olm", "channels", "list")
	if err != nil {
		return err
	}

	upgradeSequence = append(upgradeSequence, map[string]interface{}{
		"csv":    fmt.Sprintf("serverless-operator.v%s", newVersion),
		"source": "serverless-operator",
	})
	channelsList = append(channelsList, fmt.Sprintf("stable-%d.%d", newVersion.Major, newVersion.Minor))

	serving, _, _ := unstructured.NestedString(project, "dependencies", "serving")
	eventing, _, _ := unstructured.NestedString(project, "dependencies", "eventing")
	ekb, _, _ := unstructured.NestedString(project, "dependencies", "eventing_kafka_broker")

	_ = setNestedField(&node, newVersion.String(), "project", "version")
	_ = setNestedField(&node, currentVersion.String(), "olm", "replaces")
	_ = setNestedField(&node, previousVersion.String(), "olm", "previous", "replaces")
	_ = setNestedField(&node, skipRange(currentVersion, newVersion), "olm", "skipRange")
	_ = setNestedField(&node, skipRange(previousVersion, currentVersion), "olm", "previous", "skipRange")
	_ = setNestedField(&node, upgradeSequence, "upgrade_sequence")
	_ = setNestedField(&node, channelsList, "olm", "channels", "list")

	_ = setNestedField(&node, serving, "dependencies", "previous", "serving")
	_ = setNestedField(&node, eventing, "dependencies", "previous", "eventing")
	_ = setNestedField(&node, ekb, "dependencies", "previous", "eventing_kafka_broker")

	buf := bytes.NewBuffer(nil)
	if err := yaml.NewEncoder(buf).Encode(&node); err != nil {
		return fmt.Errorf("failed to encode node into buf: %w", err)
	}

	if err := os.WriteFile(projectPath, buf.Bytes(), 0600); err != nil {
		return fmt.Errorf("failed to write updates: %w", err)
	}

	return nil
}

func currentVersion(project map[string]interface{}) (*semver.Version, error) {
	v, _, err := unstructured.NestedString(project, "project", "version")
	if err != nil {
		return nil, err
	}
	ver := semver.New(v)
	return ver, nil
}

func previousVersion(project map[string]interface{}) (*semver.Version, error) {
	v, _, err := unstructured.NestedString(project, "olm", "replaces")
	if err != nil {
		return nil, err
	}
	ver := semver.New(v)
	return ver, nil
}

func skipRange(prev, curr *semver.Version) string {
	return fmt.Sprintf(">=%s <%s", prev.String(), curr.String())
}

func setNestedField(node *yaml.Node, value interface{}, fields ...string) error {

	for i, n := range node.Content {

		if i > 0 && node.Content[i-1].Value == fields[0] {

			// Base case for scalar nodes
			if len(fields) == 1 && n.Kind == yaml.ScalarNode {
				n.SetString(fmt.Sprintf("%s", value))
				break
			}
			// base case for sequence node
			if len(fields) == 1 && n.Kind == yaml.SequenceNode {

				if v, ok := value.([]interface{}); ok {
					var s yaml.Node

					b, err := yaml.Marshal(v)
					if err != nil {
						return err
					}
					if err := yaml.NewDecoder(bytes.NewBuffer(b)).Decode(&s); err != nil {
						return err
					}

					n.Content = s.Content[0].Content
				}
				break
			}

			// Continue to the next level
			return setNestedField(n, value, fields[1:]...)
		}

		if node.Kind == yaml.DocumentNode {
			return setNestedField(n, value, fields...)
		}
	}

	return nil
}
