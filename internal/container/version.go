package container

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
)

// ImageLabels represents the labels from a container image
type ImageLabels struct {
	Version  string
	Revision string
}

// GetImageLabelsFromRegistry retrieves image labels from a registry using skopeo
func GetImageLabelsFromRegistry(imageURL string) (*ImageLabels, error) {
	// Always add the docker:// prefix for skopeo
	imageURL = "docker://" + imageURL

	cmd := exec.Command("skopeo", "inspect", "--override-os", "linux", "--override-arch", "amd64", imageURL)
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to inspect image %s with skopeo: %w", imageURL, err)
	}

	var imageInfo struct {
		Labels map[string]string `json:"Labels"`
	}

	if err := json.Unmarshal(output, &imageInfo); err != nil {
		return nil, fmt.Errorf("failed to parse skopeo output: %w", err)
	}

	// Get both label sets
	jumpstarterVersion := imageInfo.Labels["jumpstarter.version"]
	jumpstarterRevision := imageInfo.Labels["jumpstarter.revision"]
	ociVersion := imageInfo.Labels["org.opencontainers.image.version"]
	ociRevision := imageInfo.Labels["org.opencontainers.image.revision"]

	// Use jumpstarter labels if both exist, otherwise fall back to OCI labels
	var version, revision string
	if jumpstarterVersion != "" && jumpstarterRevision != "" {
		version = jumpstarterVersion
		revision = jumpstarterRevision
	} else {
		version = ociVersion
		revision = ociRevision
	}

	// Only return empty labels if BOTH version and revision are completely missing
	// (neither jumpstarter nor OCI labels exist)
	return &ImageLabels{
		Version:  version,
		Revision: revision,
	}, nil
}

// GetRunningContainerLabels retrieves labels from a running container using podman inspect
func GetRunningContainerLabels(serviceName string) (*ImageLabels, error) {
	cmd := exec.Command("podman", "inspect", "--format",
		"{{index .Config.Labels \"jumpstarter.version\"}} {{index .Config.Labels \"jumpstarter.revision\"}}",
		serviceName)
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to inspect container %s: %w", serviceName, err)
	}

	parts := strings.Fields(strings.TrimSpace(string(output)))
	if len(parts) < 2 {
		return &ImageLabels{}, nil // Return empty labels if not found
	}

	return &ImageLabels{
		Version:  parts[0],
		Revision: parts[1],
	}, nil
}

// CompareVersions compares two ImageLabels and returns true if they match
func (il *ImageLabels) Matches(other *ImageLabels) bool {
	return il.Version == other.Version && il.Revision == other.Revision
}

// IsEmpty returns true if both version and revision are empty
func (il *ImageLabels) IsEmpty() bool {
	return il.Version == "" && il.Revision == ""
}

// String returns a string representation of the image labels
func (il *ImageLabels) String() string {
	if il.IsEmpty() {
		return "no version info"
	}
	return fmt.Sprintf("version=%s revision=%s", il.Version, il.Revision)
}
