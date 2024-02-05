package main

import (
	"bytes"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/coreos/go-semver/semver"
	"gopkg.in/yaml.v3"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/openshift-knative/serverless-operator/hack/cmd/common"
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
	branch := ""

	flag.StringVar(&projectPath, "project-path", projectPath, "")
	flag.StringVar(&branch, "branch", "", "")
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
	// We don't care about patch versions (patch versions are potentially skipped, etc)
	currentVersion.Patch = 0

	if branch != "" {
		majorMinor := strings.Replace(branch, "release-", "", 1)
		currentVersion = semver.New(fmt.Sprintf("%s.%d", majorMinor, 0))
	}

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
	upgradeSequence = upgradeSequence[1:] // Remove first version
	upgradeSequence = append(upgradeSequence, map[string]interface{}{
		"csv":    fmt.Sprintf("serverless-operator.v%s", newVersion),
		"source": "serverless-operator",
	})

	channelsList, _, err := unstructured.NestedSlice(project, "olm", "channels", "list")
	if err != nil {
		return err
	}
	channelsList = append(channelsList, fmt.Sprintf("stable-%d.%d", newVersion.Major, newVersion.Minor))

	serving, _, _ := unstructured.NestedString(project, "dependencies", "serving")
	eventing, _, _ := unstructured.NestedString(project, "dependencies", "eventing")
	ekb, _, _ := unstructured.NestedString(project, "dependencies", "eventing_kafka_broker")

	_ = common.SetNestedField(&node, newVersion.String(), "project", "version")
	_ = common.SetNestedField(&node, newVersion.String(), "dependencies", "redhat-knative-istio-authz-chart")
	_ = common.SetNestedField(&node, currentVersion.String(), "olm", "replaces")
	_ = common.SetNestedField(&node, previousVersion.String(), "olm", "previous", "replaces")
	_ = common.SetNestedField(&node, skipRange(currentVersion, newVersion), "olm", "skipRange")
	_ = common.SetNestedField(&node, skipRange(previousVersion, currentVersion), "olm", "previous", "skipRange")
	_ = common.SetNestedField(&node, upgradeSequence, "upgrade_sequence")
	_ = common.SetNestedField(&node, channelsList, "olm", "channels", "list")

	_ = common.SetNestedField(&node, serving, "dependencies", "previous", "serving")
	_ = common.SetNestedField(&node, eventing, "dependencies", "previous", "eventing")
	_ = common.SetNestedField(&node, ekb, "dependencies", "previous", "eventing_kafka_broker")

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
