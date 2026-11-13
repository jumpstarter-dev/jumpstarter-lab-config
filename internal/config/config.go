package config

import (
	"os"
	"path/filepath"

	"github.com/jumpstarter-dev/jumpstarter-lab-config/internal/container"
	"gopkg.in/yaml.v3"
)

// Config represents the structure of the jumpstarter-lab.yaml file.
type Config struct {
	Sources           Sources                           `yaml:"sources"`
	Instances         map[string]InstanceConfig         `yaml:"instances"`
	Variables         []string                          `yaml:"variables"`
	BaseDir           string                            `yaml:"-"` // Not serialized, set programmatically
	Loaded            *LoadedLabConfig                  `yaml:"-"` // Not serialized, used internally
	ContainerVersions map[string]*container.ImageLabels `yaml:"-"` // Not serialized, container versions by image URL
}

// Sources defines the paths for various configuration files.
type Sources struct {
	Locations            []string `yaml:"locations"`
	Clients              []string `yaml:"clients"`
	Policies             []string `yaml:"policies"`
	ExporterHosts        []string `yaml:"exporter_hosts"`
	Exporters            []string `yaml:"exporters"`
	ExporterTemplates    []string `yaml:"exporter_templates"`
	JumpstarterInstances []string `yaml:"jumpstarter_instances"`
}

// InstanceConfig overrides client and policy synchronization for a named instance.
type InstanceConfig struct {
	Clients  ResourceSyncConfig `yaml:"clients"`
	Policies ResourceSyncConfig `yaml:"policies"`
}

// ResourceSyncConfig selects whether to sync and optionally replaces global sources.
type ResourceSyncConfig struct {
	Enabled *bool    `yaml:"enabled"`
	Sources []string `yaml:"sources"` // nil inherits global sources; an empty slice selects nothing.
}

// IsEnabled honors an explicit enabled setting, then instance sources (even []),
// then global sources unless the legacy instance setting disables them.
// Without any sources, synchronization and pruning are skipped.
func (cfg ResourceSyncConfig) IsEnabled(globalSources []string, fallback *bool) bool {
	if cfg.Enabled != nil {
		return *cfg.Enabled
	}
	if cfg.Sources != nil {
		return true
	}
	return globalSources != nil && (fallback == nil || *fallback)
}

// LoadConfig reads a YAML file from the given filePath and unmarshals it into a Config struct.
func LoadConfig(filePath string, vaultPassFile string) (*Config, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	var cfg Config
	err = yaml.Unmarshal(data, &cfg)
	if err != nil {
		return nil, err
	}

	// Set the base directory containing the config file
	cfg.BaseDir = filepath.Dir(filePath)

	cfg.Loaded, err = LoadAllResources(&cfg, vaultPassFile)
	if err != nil {
		return nil, err
	}

	return &cfg, nil
}
