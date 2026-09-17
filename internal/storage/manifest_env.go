package storage

import (
	"fmt"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

// GetManifestEnv returns all environment variables defined in the manifest.
func GetManifestEnv(manifestPath string) (map[string]string, error) {
	data, err := os.ReadFile(manifestPath)
	if err != nil {
		return nil, err
	}
	var doc yaml.Node
	if err := yaml.Unmarshal(data, &doc); err != nil {
		return nil, fmt.Errorf("failed to parse manifest YAML: %w", err)
	}
	envSeq, err := getContainerEnvSeq(&doc, false)
	if err != nil || envSeq == nil {
		return make(map[string]string), nil
	}
	return extractEnvMap(envSeq), nil
}

// SetManifestEnv updates or adds environment variables in the manifest.
func SetManifestEnv(manifestPath string, envs map[string]string) error {
	data, err := os.ReadFile(manifestPath)
	if err != nil {
		return err
	}
	var doc yaml.Node
	if err := yaml.Unmarshal(data, &doc); err != nil {
		return fmt.Errorf("failed to parse manifest YAML: %w", err)
	}
	envSeq, err := getContainerEnvSeq(&doc, true)
	if err != nil {
		return err
	}
	for k, v := range envs {
		upsertEnvVar(envSeq, k, v)
	}
	return writeYamlNode(manifestPath, &doc)
}

// UnsetManifestEnv removes specified environment variables from the manifest.
func UnsetManifestEnv(manifestPath string, keys []string) error {
	data, err := os.ReadFile(manifestPath)
	if err != nil {
		return err
	}
	var doc yaml.Node
	if err := yaml.Unmarshal(data, &doc); err != nil {
		return fmt.Errorf("failed to parse manifest YAML: %w", err)
	}
	envSeq, err := getContainerEnvSeq(&doc, false)
	if err != nil || envSeq == nil {
		return nil
	}
	filterEnvSeq(envSeq, keys)
	return writeYamlNode(manifestPath, &doc)
}

// LoadDotEnv reads key-value pairs from an env file.
func LoadDotEnv(filePath string) (map[string]string, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}
	return ParseDotEnv(string(data))
}

// ParseEnvAssignments parses KEY=VALUE string slices into a map.
func ParseEnvAssignments(args []string) (map[string]string, error) {
	res := make(map[string]string, len(args))
	for _, arg := range args {
		k, v, ok := strings.Cut(arg, "=")
		if !ok || strings.TrimSpace(k) == "" {
			return nil, fmt.Errorf("invalid environment variable format %q: expected KEY=VALUE", arg)
		}
		res[strings.TrimSpace(k)] = v
	}
	return res, nil
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
	for _, k := range keys {
		keySet[k] = struct{}{}
	}
	var newContent []*yaml.Node
	for _, item := range envSeq.Content {
		nameNode := findMappingValue(item, "name")
		if nameNode != nil {
			if _, found := keySet[nameNode.Value]; found {
				continue
			}
		}
		newContent = append(newContent, item)
	}
	envSeq.Content = newContent
}
