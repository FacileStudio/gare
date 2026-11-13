package manifest

import (
	"gopkg.in/yaml.v3"
)

// GetEnv returns every environment variable defined in the manifest.
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

// SetEnv updates or adds environment variables in the manifest.
func SetEnv(manifestPath string, envs map[string]string) error {
	doc, err := readDocument(manifestPath)
	if err != nil {
		return err
	}
	container, err := containerIn(doc)
	if err != nil {
		return err
	}
	envSeq := sequenceIn(container, "env", true)
	for key, value := range envs {
		upsertEnvVar(envSeq, key, value)
	}
	return writeYAMLNode(manifestPath, doc)
}

// UnsetEnv removes the named environment variables from the manifest.
func UnsetEnv(manifestPath string, keys []string) error {
	doc, err := readDocument(manifestPath)
	if err != nil {
		return err
	}
	envSeq := envSequence(doc, false)
	if envSeq == nil {
		return nil
	}
	filterEnvSeq(envSeq, keys)
	return writeYAMLNode(manifestPath, doc)
}

// envSequence locates the container's env sequence, reporting absence as nil so callers can treat
// a manifest without one as having no environment to read or clear.
func envSequence(doc *yaml.Node, createIfMissing bool) *yaml.Node {
	container, err := containerIn(doc)
	if err != nil {
		return nil
	}
	return sequenceIn(container, "env", createIfMissing)
}

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

func filterEnvSeq(envSeq *yaml.Node, keys []string) {
	keySet := make(map[string]struct{}, len(keys))
	for _, key := range keys {
		keySet[key] = struct{}{}
	}
	var kept []*yaml.Node
	for _, item := range envSeq.Content {
		nameNode := findMappingValue(item, "name")
		if nameNode != nil {
			if _, found := keySet[nameNode.Value]; found {
				continue
			}
		}
		kept = append(kept, item)
	}
	envSeq.Content = kept
}

func upsertEnvVar(envSeq *yaml.Node, key, value string) {
	for _, item := range envSeq.Content {
		if item.Kind != yaml.MappingNode {
			continue
		}
		nameNode := findMappingValue(item, "name")
		if nameNode == nil || nameNode.Value != key {
			continue
		}
		if valNode := findMappingValue(item, "value"); valNode != nil {
			valNode.Value = value
			return
		}
		item.Content = append(item.Content,
			&yaml.Node{Kind: yaml.ScalarNode, Value: "value"},
			&yaml.Node{Kind: yaml.ScalarNode, Value: value},
		)
		return
	}
	envSeq.Content = append(envSeq.Content, &yaml.Node{
		Kind: yaml.MappingNode,
		Content: []*yaml.Node{
			{Kind: yaml.ScalarNode, Value: "name"},
			{Kind: yaml.ScalarNode, Value: key},
			{Kind: yaml.ScalarNode, Value: "value"},
			{Kind: yaml.ScalarNode, Value: value},
		},
	})
}
