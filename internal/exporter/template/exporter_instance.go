package template

import (
	"fmt"

	v1alpha1 "github.com/jumpstarter-dev/jumpstarter-lab-config/api/v1alpha1"
	"github.com/jumpstarter-dev/jumpstarter-lab-config/internal/config"
	"github.com/jumpstarter-dev/jumpstarter-lab-config/internal/templating"
)

type ExporterInstanceTemplater struct {
	config                 *config.Config
	exporterInstance       *v1alpha1.ExporterInstance
	exporterConfigTemplate *v1alpha1.ExporterConfigTemplate
}

func NewExporterInstanceTemplate(cfg *config.Config, exporterInstance *v1alpha1.ExporterInstance) (*ExporterInstanceTemplater, error) {
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
		return nil, nil, fmt.Errorf("error applying template %w", err)
	}

	templateParametersMap := exporterInstanceCopy.Spec.ConfigTemplateRef.Parameters
	templateParameters := templating.NewParameters("exporter-instance")
	templateParameters.SetFromMap(templateParametersMap)

	exporterConfigTemplateCopy := e.exporterConfigTemplate.DeepCopy()
	err = tapplier.ApplyWithParameters(exporterConfigTemplateCopy, templateParameters)
	if err != nil {
		return nil, nil, fmt.Errorf("error applying template %w", err)
	}

	return exporterInstanceCopy, exporterConfigTemplateCopy, nil
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

func (e *ExporterInstanceTemplater) RenderTemplateConfig() (string, error) {
	_, exporterConfigTemplateCopy, err := e.renderTemplates()
	if err != nil {
		return "", err
	}

	return exporterConfigTemplateCopy.Spec.ConfigTemplate, nil
}
