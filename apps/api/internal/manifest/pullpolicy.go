package manifest

import (
	"strings"

	"gopkg.in/yaml.v3"
)

const (
	localImagePrefix     = "localhost/"
	imageKey             = "image"
	imagePullPolicyKey   = "imagePullPolicy"
	localImagePullPolicy = "Never"
)

// EnsureLocalImagePullPolicy pins imagePullPolicy on every container whose image gare builds and
// tags locally. A gare-built image only exists in local storage, so podman must never be asked to
// resolve it from a registry: an unset policy leaves a missing or reloaded image to be looked up as
// docker://localhost/<name>, which fails wherever nothing answers as a registry on localhost.
func EnsureLocalImagePullPolicy(manifestPath string) error {
	doc, err := readDocument(manifestPath)
	if err != nil {
		return err
	}
	changed := false
	for _, container := range allContainers(doc) {
		if pinLocalPullPolicy(container) {
			changed = true
		}
	}
	if !changed {
		return nil
	}
	return writeYAMLNode(manifestPath, doc)
}

// allContainers returns every container declared under the manifest's spec, which is the set of
// images a pod resolves when it starts.
func allContainers(doc *yaml.Node) []*yaml.Node {
	if doc.Kind != yaml.DocumentNode || len(doc.Content) == 0 {
		return nil
	}
	spec := findMappingValue(doc.Content[0], "spec")
	if spec == nil {
		return nil
	}
	containers := findMappingValue(spec, "containers")
	if containers == nil || containers.Kind != yaml.SequenceNode {
		return nil
	}
	return containers.Content
}

// pinLocalPullPolicy adds the local pull policy to one container, leaving a policy the manifest
// already states, and any image gare did not build, untouched.
func pinLocalPullPolicy(container *yaml.Node) bool {
	if container.Kind != yaml.MappingNode {
		return false
	}
	image := findMappingValue(container, imageKey)
	if image == nil || !strings.HasPrefix(image.Value, localImagePrefix) {
		return false
	}
	if findMappingValue(container, imagePullPolicyKey) != nil {
		return false
	}
	container.Content = append(container.Content,
		&yaml.Node{Kind: yaml.ScalarNode, Value: imagePullPolicyKey},
		&yaml.Node{Kind: yaml.ScalarNode, Value: localImagePullPolicy},
	)
	return true
}
