package manifest

import (
	"fmt"
	"os"
	"strconv"

	"gopkg.in/yaml.v3"
)

// UpdatePorts rewrites containerPort and hostPort in an existing manifest.
func UpdatePorts(manifestPath string, containerPort, hostPort int) error {
	if hostPort <= 0 || hostPort > 65535 {
		return fmt.Errorf("hostPort must be between 1 and 65535, got %d", hostPort)
	}
	if containerPort > 65535 {
		return fmt.Errorf("containerPort must be between 1 and 65535, got %d", containerPort)
	}
	doc, err := readDocument(manifestPath)
	if err != nil {
		return err
	}
	container, err := containerIn(doc)
	if err != nil {
		return fmt.Errorf("failed to locate container ports: %w", err)
	}
	ports := sequenceIn(container, "ports", true)
	if len(ports.Content) == 0 {
		ports.Content = append(ports.Content, &yaml.Node{Kind: yaml.MappingNode})
	}
	updatePortMapping(ports.Content[0], containerPort, hostPort)
	return writeYAMLNode(manifestPath, doc)
}

func readDocument(path string) (*yaml.Node, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var doc yaml.Node
	if err := yaml.Unmarshal(data, &doc); err != nil {
		return nil, fmt.Errorf("failed to parse manifest YAML: %w", err)
	}
	return &doc, nil
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
