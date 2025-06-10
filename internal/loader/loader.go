package loader

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"

	jsApi "github.com/jumpstarter-dev/jumpstarter-controller/api/v1alpha1"
	api "github.com/jumpstarter-dev/jumpstarter-lab-config/api/v1alpha1"
	"github.com/jumpstarter-dev/jumpstarter-lab-config/internal/config"
	"github.com/jumpstarter-dev/jumpstarter-lab-config/internal/vars"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
)

// LoadedLabConfig holds all unmarshalled resources from the configuration.
// Resources are stored in maps keyed by their metadata.name.
type LoadedLabConfig struct {
	Clients                 map[string]*jsApi.Client
	Policies                map[string]*jsApi.ExporterAccessPolicy
	PhysicalLocations       map[string]*api.PhysicalLocation
	ExporterHosts           map[string]*api.ExporterHost
	ExporterInstances       map[string]*api.ExporterInstance
	ExporterConfigTemplates map[string]*api.ExporterConfigTemplate
	JumpstarterInstances    map[string]*api.JumpstarterInstance
	Variables               *vars.Variables // Variables loaded from the config

	// SourceFiles tracks which file each resource was loaded from
	// Format: SourceFiles[objectType][objectName] = filename
	SourceFiles map[string]map[string]string
}

var (
	scheme       = runtime.NewScheme()
	codecFactory serializer.CodecFactory
)

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	// Register types from your local api/v1alpha1 package
	utilruntime.Must(api.AddToScheme(scheme))
	utilruntime.Must(jsApi.AddToScheme(scheme))

	codecFactory = serializer.NewCodecFactory(scheme, serializer.EnableStrict)
}

// readAndDecodeYAMLFile reads a YAML file and decodes it into a runtime.Object.
func readAndDecodeYAMLFile(filePath string) (runtime.Object, error) {
	yamlFile, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("error reading YAML file %s: %w", filePath, err)
	}
	decode := codecFactory.UniversalDeserializer().Decode
	obj, gvk, err := decode(yamlFile, nil, nil)
	if err != nil {
		return nil, fmt.Errorf("error decoding YAML from file %s (GVK: %v): %w", filePath, gvk, err)
	}
	return obj, nil
}

// processResourceGlobs finds files matching a list of glob patterns, decodes them,
// and stores them in the provided targetMap.
// targetMap must be a pointer to a map (e.g., &loadedCfg.PhysicalLocations).
// resourceTypeName is used for logging and error messages.
// cfg contains the base directory to resolve relative paths against.
// sourceFiles is used to track which file each resource was loaded from.
func processResourceGlobs(globPatterns []string, targetMap interface{}, resourceTypeName string, cfg *config.Config, sourceFiles map[string]map[string]string) error {
	if len(globPatterns) == 0 {
		return nil // Skip if no glob patterns are provided
	}

	var allFilePaths []string
	for _, globPattern := range globPatterns {
		if globPattern == "" {
			continue // Skip empty patterns
		}

		// Resolve the glob pattern relative to the config directory
		absoluteGlobPattern := filepath.Join(cfg.BaseDir, globPattern)
		filePaths, err := filepath.Glob(absoluteGlobPattern)
		if err != nil {
			return fmt.Errorf("processResourceGlobs: error evaluating glob pattern '%s' for %s: %w", globPattern, resourceTypeName, err)
		}
		allFilePaths = append(allFilePaths, filePaths...)
	}

	mapVal := reflect.ValueOf(targetMap).Elem()  // .Elem() because targetMap is a pointer to the map
	expectedMapValueType := mapVal.Type().Elem() // e.g., *api.PhysicalLocation

	for _, filePath := range allFilePaths {
		obj, err := readAndDecodeYAMLFile(filePath)
		if err != nil {
			// Stop at first error encountered
			return fmt.Errorf("processResourceGlob: error processing file %s for %s: %w", filePath, resourceTypeName, err)
		}

		metaObj, ok := obj.(metav1.Object)
		if !ok {
			return fmt.Errorf("processResourceGlob: object from file %s (%T) does not implement metav1.Object, expected for %s", filePath, obj, resourceTypeName)
		}
		name := metaObj.GetName()
		if name == "" {
			return fmt.Errorf("processResourceGlob: object from file %s for %s is missing metadata.name", filePath, resourceTypeName)
		}

		objValue := reflect.ValueOf(obj)
		if !objValue.Type().AssignableTo(expectedMapValueType) {
			return fmt.Errorf("processResourceGlobs: file %s (name: %s) decoded to type %T, but expected assignable to %s for %s map", filePath, name, obj, expectedMapValueType, resourceTypeName)
		}

		if mapVal.MapIndex(reflect.ValueOf(name)).IsValid() {
			// Find the original file that contained this duplicate name
			originalFile := sourceFiles[resourceTypeName][name]
			return fmt.Errorf("processResourceGlobs: duplicate %s name: '%s' found in file %s (originally defined in %s)", resourceTypeName, name, filePath, originalFile)
		}

		// Track the source file for this resource
		if sourceFiles[resourceTypeName] == nil {
			sourceFiles[resourceTypeName] = make(map[string]string)
		}
		sourceFiles[resourceTypeName][name] = filePath

		mapVal.SetMapIndex(reflect.ValueOf(name), objValue)
	}
	return nil
}

// LoadAllResources processes the configuration sources, loads all specified YAML files,
// unmarshals them into their respective API types, and returns a LoadedLabConfig struct.
func LoadAllResources(cfg *config.Config, vaultPassFile string) (*LoadedLabConfig, error) {

	variables, err := vars.NewVariables(vaultPassFile)
	if err != nil {
		return nil, fmt.Errorf("LoadAllResources: failed to create Variables instance: %w", err)
	}

	loaded := &LoadedLabConfig{
		Clients:                 make(map[string]*jsApi.Client),
		Policies:                make(map[string]*jsApi.ExporterAccessPolicy),
		PhysicalLocations:       make(map[string]*api.PhysicalLocation),
		ExporterHosts:           make(map[string]*api.ExporterHost),
		ExporterInstances:       make(map[string]*api.ExporterInstance),
		ExporterConfigTemplates: make(map[string]*api.ExporterConfigTemplate),
		JumpstarterInstances:    make(map[string]*api.JumpstarterInstance),
		SourceFiles:             make(map[string]map[string]string),
		Variables:               variables,
	}

	type sourceMapping struct {
		globPatterns     []string
		targetMap        interface{}
		resourceTypeName string
	}

	mappings := []sourceMapping{
		{cfg.Sources.Clients, &loaded.Clients, "Client"},
		{cfg.Sources.Policies, &loaded.Policies, "ExporterAccessPolicy"},
		{cfg.Sources.Locations, &loaded.PhysicalLocations, "PhysicalLocation"},
		{cfg.Sources.ExporterHosts, &loaded.ExporterHosts, "ExporterHost"},
		{cfg.Sources.Exporters, &loaded.ExporterInstances, "ExporterInstance"},
		{cfg.Sources.ExporterTemplates, &loaded.ExporterConfigTemplates, "ExporterConfigTemplate"},
		{cfg.Sources.JumpstarterInstances, &loaded.JumpstarterInstances, "JumpstarterInstance"},
	}

	for _, m := range mappings {
		if err := processResourceGlobs(m.globPatterns, m.targetMap, m.resourceTypeName, cfg, loaded.SourceFiles); err != nil {
			return nil, fmt.Errorf("failed to load %s: %w", m.resourceTypeName, err)
		}
	}

	for _, filePath := range cfg.Variables {
		// calculate filepath based on the config's base directory
		baseDirPath := filepath.Join(cfg.BaseDir, filePath)
		fmt.Println("Loading variables from:", baseDirPath)
		if err := variables.LoadFromFile(baseDirPath); err != nil {
			return nil, fmt.Errorf("LoadAllResources: error loading variables from file %s: %w", filePath, err)
		}
	}

	return loaded, nil
}
