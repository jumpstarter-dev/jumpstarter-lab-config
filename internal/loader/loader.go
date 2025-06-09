package loader

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"

	api "github.com/jumpstarter-dev/jumpstarter-lab-config/api/v1alpha1"
	"github.com/jumpstarter-dev/jumpstarter-lab-config/internal/config"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
)

// LoadedLabConfig holds all unmarshalled resources from the configuration.
// Resources are stored in maps keyed by their metadata.name.
type LoadedLabConfig struct {
	PhysicalLocations       map[string]*api.PhysicalLocation
	ExporterHosts           map[string]*api.ExporterHost
	ExporterInstances       map[string]*api.ExporterInstance
	ExporterConfigTemplates map[string]*api.ExporterConfigTemplate
	JumpstarterInstances    map[string]*api.JumpstarterInstance

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

	codecFactory = serializer.NewCodecFactory(scheme, serializer.EnableStrict)
}

// setSourceFile sets the sourceFile field on an object if it exists and is settable.
func setSourceFile(obj runtime.Object, filePath string) error {
	objValue := reflect.ValueOf(obj)

	// Handle pointer to struct
	if objValue.Kind() == reflect.Ptr && objValue.Elem().Kind() == reflect.Struct {
		sourceFileField := objValue.Elem().FieldByName("sourceFile")
		if sourceFileField.IsValid() && sourceFileField.CanSet() && sourceFileField.Kind() == reflect.String {
			sourceFileField.SetString(filePath)
			return nil
		}
	}

	// Handle struct directly (though less common in Kubernetes objects)
	if objValue.Kind() == reflect.Struct {
		sourceFileField := objValue.FieldByName("sourceFile")
		if sourceFileField.IsValid() && sourceFileField.CanSet() && sourceFileField.Kind() == reflect.String {
			sourceFileField.SetString(filePath)
			return nil
		}
	}

	// sourceFile field not found or not settable - this is not an error
	return nil
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

		// Set the sourceFile field if it exists
		if err := setSourceFile(obj, filePath); err != nil {
			return fmt.Errorf("processResourceGlobs: failed to set sourceFile for %s from file %s: %w", resourceTypeName, filePath, err)
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
func LoadAllResources(cfg *config.Config) (*LoadedLabConfig, error) {
	loaded := &LoadedLabConfig{
		PhysicalLocations:       make(map[string]*api.PhysicalLocation),
		ExporterHosts:           make(map[string]*api.ExporterHost),
		ExporterInstances:       make(map[string]*api.ExporterInstance),
		ExporterConfigTemplates: make(map[string]*api.ExporterConfigTemplate),
		JumpstarterInstances:    make(map[string]*api.JumpstarterInstance),
		SourceFiles:             make(map[string]map[string]string),
	}

	type sourceMapping struct {
		globPatterns     []string
		targetMap        interface{}
		resourceTypeName string
	}

	mappings := []sourceMapping{
		{cfg.Sources.Locations, &loaded.PhysicalLocations, "PhysicalLocation"},
		{cfg.Sources.ExporterHosts, &loaded.ExporterHosts, "ExporterHost"},
		{cfg.Sources.Exporters, &loaded.ExporterInstances, "ExporterInstance"}, // Assuming Sources.Exporters maps to ExporterInstance
		{cfg.Sources.ExporterTemplates, &loaded.ExporterConfigTemplates, "ExporterConfigTemplate"},
		{cfg.Sources.JumpstarterInstances, &loaded.JumpstarterInstances, "JumpstarterInstance"},
	}

	for _, m := range mappings {
		if err := processResourceGlobs(m.globPatterns, m.targetMap, m.resourceTypeName, cfg, loaded.SourceFiles); err != nil {
			return nil, fmt.Errorf("failed to load %s: %w", m.resourceTypeName, err)
		}
	}

	return loaded, nil
}
