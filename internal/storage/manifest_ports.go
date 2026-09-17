package storage

import (
	"fmt"
	"os"
	"strconv"

	"gopkg.in/yaml.v3"
)

// UpdateManifestPorts updates containerPort and hostPort in an existing manifest.
func UpdateManifestPorts(manifestPath string, containerPort, hostPort int) error {
	if hostPort <= 0 || hostPort > 65535 {
		return fmt.Errorf("hostPort must be between 1 and 65535, got %d", hostPort)
	}
	if containerPort > 65535 {
		return fmt.Errorf("containerPort must be between 1 and 65535, got %d", containerPort)
	}
	data, err := os.ReadFile(manifestPath)
	if err != nil {
		return err
	}
	var doc yaml.Node
	if err := yaml.Unmarshal(data, &doc); err != nil {
		return fmt.Errorf("failed to parse manifest YAML: %w", err)
	}
	portsSeq, err := getContainerPortsSeq(&doc, true)
	if err != nil {
		return fmt.Errorf("failed to locate container ports: %w", err)
	}
	if len(portsSeq.Content) == 0 {
		portsSeq.Content = append(portsSeq.Content, &yaml.Node{Kind: yaml.MappingNode})
	}
	updatePortMapping(portsSeq.Content[0], containerPort, hostPort)
	return writeYamlNode(manifestPath, &doc)
}

func getContainerPortsSeq(doc *yaml.Node, createIfMissing bool) (*yaml.Node, error) {
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
	portsSeq := findMappingValue(container, "ports")
	if portsSeq == nil && createIfMissing {
		keyNode := &yaml.Node{Kind: yaml.ScalarNode, Value: "ports"}
		portsSeq = &yaml.Node{Kind: yaml.SequenceNode, Content: []*yaml.Node{}}
		container.Content = append(container.Content, keyNode, portsSeq)
	}
	return portsSeq, nil
}

func updatePortMapping(portNode *yaml.Node, containerPort, hostPort int) {
	if portNode.Kind != yaml.MappingNode {
		return
	}
	upsertPortKey(portNode, "containerPort", containerPort)
	upsertPortKey(portNode, "hostPort", hostPort)
}

func upsertPortKey(portNode *yaml.Node, key string, port int) {
	if port <= 0 {
		return
	}
	for i := 0; i < len(portNode.Content)-1; i += 2 {
		if portNode.Content[i].Value == key {
			portNode.Content[i+1].Value = strconv.Itoa(port)
			return
		}
	}
	portNode.Content = append(portNode.Content,
		&yaml.Node{Kind: yaml.ScalarNode, Value: key},
		&yaml.Node{Kind: yaml.ScalarNode, Value: strconv.Itoa(port)},
	)
}
