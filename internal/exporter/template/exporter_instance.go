package template

import (
	"fmt"

	v1alpha1 "github.com/jumpstarter-dev/jumpstarter-lab-config/api/v1alpha1"
	"github.com/jumpstarter-dev/jumpstarter-lab-config/internal/config"
	"github.com/jumpstarter-dev/jumpstarter-lab-config/internal/templating"
)

type ServiceParameters struct {
	TlsCA string
	Token string
}
type ExporterInstanceTemplater struct {
	config                 *config.Config
	exporterInstance       *v1alpha1.ExporterInstance
	exporterConfigTemplate *v1alpha1.ExporterConfigTemplate
	serviceParameters      ServiceParameters
}

func NewExporterInstanceTemplater(cfg *config.Config, exporterInstance *v1alpha1.ExporterInstance) (*ExporterInstanceTemplater, error) {
	exporterConfigTemplate, ok := cfg.Loaded.ExporterConfigTemplates[exporterInstance.Spec.ConfigTemplateRef.Name]
	if !ok {
		return nil, fmt.Errorf("exporter config template %s not found", exporterInstance.Spec.ConfigTemplateRef.Name)
	}
	return &ExporterInstanceTemplater{
		config:                 cfg,
		exporterInstance:       exporterInstance,
		exporterConfigTemplate: exporterConfigTemplate,
	}, nil
}

func (e *ExporterInstanceTemplater) SetServiceParameters(serviceParameters ServiceParameters) {
	e.serviceParameters = serviceParameters
}

func (s *ServiceParameters) Parameters() *templating.Parameters {
	parameters := templating.NewParameters("service")
	parameters.Set("tls_ca", s.TlsCA)
	parameters.Set("token", s.Token)
	return parameters
}

// renderTemplates applies templates to both the exporterInstance and exporterConfigTemplate
// and returns the rendered copies
func (e *ExporterInstanceTemplater) renderTemplates() (*v1alpha1.ExporterInstance, *v1alpha1.ExporterConfigTemplate, error) {
	tapplier, err := templating.NewTemplateApplier(e.config, nil)
	if err != nil {
		return nil, nil, fmt.Errorf("error creating template applier %w", err)
	}

	exporterInstanceCopy := e.exporterInstance.DeepCopy()
	err = tapplier.Apply(exporterInstanceCopy)
	if err != nil {
		return nil, nil, fmt.Errorf("ExporterInstance: %w", err)
	}

	namespace, endpoint, err := e.GetNamespaceAndEndpoint()
	if err != nil {
		return nil, nil, err
	}

	templateParametersMap := exporterInstanceCopy.Spec.ConfigTemplateRef.Parameters
	templateParametersMap["namespace"] = namespace
	templateParametersMap["endpoint"] = endpoint
	templateParametersMap["container_image"] = e.exporterConfigTemplate.Spec.ContainerImage
	templateParameters := templating.NewParameters("exporter-instance")
	templateParameters.SetFromMap(templateParametersMap)

	exporterConfigTemplateCopy := e.exporterConfigTemplate.DeepCopy()

	err = tapplier.ApplyWithParameters(exporterConfigTemplateCopy,
		templateParameters.Merge(e.serviceParameters.Parameters()))

	if err != nil {
		return nil, nil, fmt.Errorf("ExporterConfigTemplate: %w", err)
	}

	return exporterInstanceCopy, exporterConfigTemplateCopy, nil
}

func (e *ExporterInstanceTemplater) GetNamespaceAndEndpoint() (string, string, error) {
	jsJumpstarterInstanceName := e.exporterInstance.Spec.JumpstarterInstanceRef.Name
	jsJumpstarterInstance, ok := e.config.Loaded.GetJumpstarterInstances()[jsJumpstarterInstanceName]
	if !ok {
		return "", "", fmt.Errorf("in ExporterInstance %s: jumpstarter instance %s not found", e.exporterInstance.Name, jsJumpstarterInstanceName)
	}
	if len(jsJumpstarterInstance.Spec.Endpoints) == 0 {
		return "", "", fmt.Errorf("in ExporterInstance %s: jumpstarter instance %s has no endpoints", e.exporterInstance.Name, jsJumpstarterInstanceName)
	}
	return jsJumpstarterInstance.Spec.Namespace, jsJumpstarterInstance.Spec.Endpoints[0], nil
}

func (e *ExporterInstanceTemplater) RenderTemplateLabels() (map[string]string, error) {
	exporterInstanceCopy, exporterConfigTemplateCopy, err := e.renderTemplates()
	if err != nil {
		return nil, err
	}

	// merge labels in exporterConfigTemplateCopy.Spec.ExporterMetadata.Labels with exporterInstance.Labels
	labels := make(map[string]string)
	for key, value := range exporterConfigTemplateCopy.Spec.ExporterMetadata.Labels {
		labels[key] = value
	}
	for key, value := range exporterInstanceCopy.Labels {
		labels[key] = value
	}

	return labels, nil
}

func (e *ExporterInstanceTemplater) RenderTemplateConfig() (*v1alpha1.ExporterConfigTemplate, error) {
	_, exporterConfigTemplateCopy, err := e.renderTemplates()
	if err != nil {
		return nil, err
	}

	return exporterConfigTemplateCopy, nil
}
