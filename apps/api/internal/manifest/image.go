package manifest

import (
	"strings"
)

// SetImage points every container that runs an image of repository repo, whatever its tag, at ref.
// It is how a deploy pins the pod to the image it just built and how a rollback re-points it at an
// earlier one, leaving every other image the manifest names untouched.
func SetImage(manifestPath, repo, ref string) error {
	doc, err := readDocument(manifestPath)
	if err != nil {
		return err
	}
	changed := false
	for _, container := range allContainers(doc) {
		image := findMappingValue(container, imageKey)
		if image == nil || image.Value == ref || imageRepository(image.Value) != repo {
			continue
		}
		image.Value = ref
		changed = true
	}
	if !changed {
		return nil
	}
	return writeYAMLNode(manifestPath, doc)
}

// imageRepository strips the tag from an image reference. The tag follows the last colon after the
// last slash, so a registry port is not mistaken for one.
func imageRepository(image string) string {
	slash := strings.LastIndex(image, "/")
	if colon := strings.LastIndex(image, ":"); colon > slash {
		return image[:colon]
	}
	return image
}
