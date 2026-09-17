package storage

import (
	"bytes"
	"fmt"

	"github.com/FacileStudio/gare/internal/atomicfile"
	"gopkg.in/yaml.v3"
)

func writeYamlNode(filePath string, doc *yaml.Node) error {
	var buf bytes.Buffer
	enc := yaml.NewEncoder(&buf)
	enc.SetIndent(2)
	encodeErr := enc.Encode(doc)
	closeErr := enc.Close()
	if encodeErr != nil {
		return encodeErr
	}
	if closeErr != nil {
		return closeErr
	}
	return atomicfile.WriteFile(filePath, buf.Bytes(), 0644)
}

func findMappingValue(node *yaml.Node, key string) *yaml.Node {
	if node.Kind != yaml.MappingNode {
		return nil
	}
	for i := 0; i < len(node.Content)-1; i += 2 {
		if node.Content[i].Value == key {
			return node.Content[i+1]
		}
	}
	return nil
}

func getContainerEnvSeq(doc *yaml.Node, createIfMissing bool) (*yaml.Node, error) {
	if doc.Kind != yaml.DocumentNode || len(doc.Content) == 0 {
		return nil, fmt.Errorf("invalid manifest document")
	}
	root := doc.Content[0]
	spec := findMappingValue(root, "spec")
	if spec == nil {
		return nil, fmt.Errorf("missing spec in manifest")
	}
	containers := findMappingValue(spec, "containers")
	if containers == nil || containers.Kind != yaml.SequenceNode || len(containers.Content) == 0 {
		return nil, fmt.Errorf("missing containers in manifest spec")
	}
	container := containers.Content[0]
	envSeq := findMappingValue(container, "env")
	if envSeq == nil && createIfMissing {
		keyNode := &yaml.Node{Kind: yaml.ScalarNode, Value: "env"}
		envSeq = &yaml.Node{Kind: yaml.SequenceNode, Content: []*yaml.Node{}}
		container.Content = append(container.Content, keyNode, envSeq)
	}
	return envSeq, nil
}

func upsertEnvVar(envSeq *yaml.Node, k, v string) {
	for _, item := range envSeq.Content {
		if item.Kind != yaml.MappingNode {
			continue
		}
		nameNode := findMappingValue(item, "name")
		if nameNode != nil && nameNode.Value == k {
			valNode := findMappingValue(item, "value")
			if valNode != nil {
				valNode.Value = v
				return
			}
			item.Content = append(item.Content,
				&yaml.Node{Kind: yaml.ScalarNode, Value: "value"},
				&yaml.Node{Kind: yaml.ScalarNode, Value: v},
			)
			return
		}
	}
	item := &yaml.Node{
		Kind: yaml.MappingNode,
		Content: []*yaml.Node{
			{Kind: yaml.ScalarNode, Value: "name"},
			{Kind: yaml.ScalarNode, Value: k},
			{Kind: yaml.ScalarNode, Value: "value"},
			{Kind: yaml.ScalarNode, Value: v},
		},
	}
	envSeq.Content = append(envSeq.Content, item)
}
