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
	chart := make(map[string]interface{}, 8)
	wd, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}
	chartMetadataPath := filepath.Join(wd, "Chart.yaml")
	branch := ""

	flag.StringVar(&chartMetadataPath, "chart-metadata-path", chartMetadataPath, "")
	flag.StringVar(&branch, "branch", "", "")
	flag.Parse()

	var node yaml.Node

	file, err := os.ReadFile(chartMetadataPath)
	if err != nil {
		return fmt.Errorf("failed to read file %s: %w", chartMetadataPath, err)
	}

	if err := yaml.NewDecoder(bytes.NewBuffer(file)).Decode(&chart); err != nil {
		return fmt.Errorf("failed to decode file into map: %w", err)
	}
	if err := yaml.NewDecoder(bytes.NewBuffer(file)).Decode(&node); err != nil {
		return fmt.Errorf("failed to decode file into node: %w", err)
	}

	currentVersion, err := currentVersion(chart)
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

	_ = common.SetNestedField(&node, newVersion.String(), "version")
	_ = common.SetNestedField(&node, newVersion.String(), "appVersion")

	buf := bytes.NewBuffer(nil)
	if err := yaml.NewEncoder(buf).Encode(&node); err != nil {
		return fmt.Errorf("failed to encode node into buf: %w", err)
	}

	if err := os.WriteFile(chartMetadataPath, buf.Bytes(), 0600); err != nil {
		return fmt.Errorf("failed to write updates: %w", err)
	}

	return nil
}

func currentVersion(project map[string]interface{}) (*semver.Version, error) {
	v, _, err := unstructured.NestedString(project, "version")
	if err != nil {
		return nil, err
	}
	ver := semver.New(v)
	return ver, nil
}
