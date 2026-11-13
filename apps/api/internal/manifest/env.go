package manifest

import (
	"errors"
	"io/fs"
	"maps"
	"slices"

	"gopkg.in/yaml.v3"
)

// GetEnv returns every environment variable the manifest's first container sets by value.
func GetEnv(manifestPath string) (map[string]string, error) {
	doc, err := readDocument(manifestPath)
	if err != nil {
		return nil, err
	}
	envSeq := envSequence(doc, false)
	if envSeq == nil {
		return map[string]string{}, nil
	}
	return extractEnvMap(envSeq), nil
}

// RenderEnv sets the first container's env to the entries the repository's manifest declares, read
// from baseManifestPath, overlaid with envs. Starting from the repository's entries rather than from
// what the manifest holds now is what lets a variable removed from the app env file leave the pod,
// while one the repository declares stays. The app env file wins a name clash. A baseManifestPath
// that is empty or names no file means the repository declares nothing.
func RenderEnv(manifestPath, baseManifestPath string, envs map[string]string) error {
	base, err := baseEnvEntries(baseManifestPath)
	if err != nil {
		return err
	}
	doc, err := readDocument(manifestPath)
	if err != nil {
		return err
	}
	if len(envs)+len(base) == 0 && envSequence(doc, false) == nil {
		return nil
	}
	container, err := containerIn(doc)
	if err != nil {
		return err
	}
	envSeq := sequenceIn(container, "env", true)
	envSeq.Content = base
	for _, k := range slices.Sorted(maps.Keys(envs)) {
		upsertEnvVar(envSeq, k, envs[k])
	}
	return writeYAMLNode(manifestPath, doc)
}

// envSequence locates the first container's env sequence, reporting a manifest without a container
// or without env as nil, which callers read as no environment at all.
func envSequence(doc *yaml.Node, createIfMissing bool) *yaml.Node {
	container, err := containerIn(doc)
	if err != nil {
		return nil
	}
	return sequenceIn(container, "env", createIfMissing)
}

// baseEnvEntries returns the env entries a repository's manifest declares, valueFrom entries
// included, so rendering keeps everything the repository asked for.
func baseEnvEntries(path string) ([]*yaml.Node, error) {
	if path == "" {
		return nil, nil
	}
	doc, err := readDocument(path)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if envSeq := envSequence(doc, false); envSeq != nil {
		return envSeq.Content, nil
	}
	return nil, nil
}

// extractEnvMap reads the name and literal value of each env entry, skipping valueFrom entries,
// which hold no value gare could move or show.
func extractEnvMap(envSeq *yaml.Node) map[string]string {
	res := make(map[string]string)
	for _, item := range envSeq.Content {
		if item.Kind != yaml.MappingNode {
			continue
		}
		nameNode := findMappingValue(item, "name")
		valNode := findMappingValue(item, "value")
		if nameNode != nil && valNode != nil {
			res[nameNode.Value] = valNode.Value
		}
	}
	return res
}

// upsertEnvVar sets one variable, replacing an entry of the same name wholesale so a repository's
// valueFrom for that name cannot shadow the operator's value. The value is tagged as a string, so
// `true` or `8080` reach the pod as text rather than as a YAML bool or int the pod spec rejects.
func upsertEnvVar(envSeq *yaml.Node, key, value string) {
	entry := &yaml.Node{
		Kind: yaml.MappingNode,
		Content: []*yaml.Node{
			{Kind: yaml.ScalarNode, Value: "name"},
			{Kind: yaml.ScalarNode, Value: key},
			{Kind: yaml.ScalarNode, Value: "value"},
			{Kind: yaml.ScalarNode, Tag: "!!str", Value: value},
		},
	}
	for i, item := range envSeq.Content {
		if nameNode := findMappingValue(item, "name"); nameNode != nil && nameNode.Value == key {
			envSeq.Content[i] = entry
			return
		}
	}
	envSeq.Content = append(envSeq.Content, entry)
}
