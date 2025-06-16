package config

import (
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// Config represents the structure of the jumpstarter-lab.yaml file.
type Config struct {
	Sources   Sources          `yaml:"sources"`
	Variables []string         `yaml:"variables"`
	BaseDir   string           `yaml:"-"` // Not serialized, set programmatically
	Loaded    *LoadedLabConfig `yaml:"-"` // Not serialized, used internally
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

	return &cfg, err
}
