// Package manifest reads and rewrites the Kubernetes Pod manifest gare supervises.
package manifest

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"text/template"

	"github.com/FacileStudio/gare/internal/atomicfile"
	"gopkg.in/yaml.v3"
)

var defaultTemplate = template.Must(template.New("manifest").Parse(`apiVersion: v1
kind: Pod
metadata:
  name: {{.Name}}
  labels:
    app: {{.Name}}
spec:
  containers:
  - name: {{.Name}}
    image: localhost/{{.Name}}:latest
    imagePullPolicy: Never
    ports:
    - containerPort: {{.ContainerPort}}
      hostPort: {{.HostPort}}
`))

// Generate writes a default Pod manifest mapping containerPort to hostPort.
func Generate(name string, containerPort int, hostPort int, destPath string) error {
	if hostPort <= 0 || hostPort > 65535 {
		return fmt.Errorf("hostPort must be between 1 and 65535, got %d", hostPort)
	}
	if containerPort <= 0 {
		containerPort = hostPort
	}
	if containerPort > 65535 {
		return fmt.Errorf("containerPort must be between 1 and 65535, got %d", containerPort)
	}
	targetFile := destPath
	if fi, err := os.Stat(destPath); (err == nil && fi.IsDir()) || filepath.Ext(destPath) == "" {
		targetFile = filepath.Join(destPath, "manifest.yaml")
	}
	var buf bytes.Buffer
	data := struct {
		Name          string
		ContainerPort int
		HostPort      int
	}{
		Name:          name,
		ContainerPort: containerPort,
		HostPort:      hostPort,
	}
	if err := defaultTemplate.Execute(&buf, data); err != nil {
		return fmt.Errorf("failed to execute manifest template: %w", err)
	}
	return atomicfile.WriteFile(targetFile, buf.Bytes(), 0644)
}

func writeYAMLNode(filePath string, doc *yaml.Node) error {
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

func containerIn(doc *yaml.Node) (*yaml.Node, error) {
	if doc.Kind != yaml.DocumentNode || len(doc.Content) == 0 {
		return nil, fmt.Errorf("invalid manifest document")
	}
	spec := findMappingValue(doc.Content[0], "spec")
	if spec == nil {
		return nil, fmt.Errorf("missing spec in manifest")
	}
	containers := findMappingValue(spec, "containers")
	if containers == nil || containers.Kind != yaml.SequenceNode || len(containers.Content) == 0 {
		return nil, fmt.Errorf("missing containers in manifest spec")
	}
	return containers.Content[0], nil
}

func sequenceIn(container *yaml.Node, key string, createIfMissing bool) *yaml.Node {
	seq := findMappingValue(container, key)
	if seq == nil && createIfMissing {
		seq = &yaml.Node{Kind: yaml.SequenceNode, Content: []*yaml.Node{}}
		container.Content = append(container.Content,
			&yaml.Node{Kind: yaml.ScalarNode, Value: key}, seq)
	}
	return seq
}
