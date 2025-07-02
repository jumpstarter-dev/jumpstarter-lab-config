package templating

import "github.com/jumpstarter-dev/jumpstarter-lab-config/internal/output"

// define a Parameters struct that holds additional parameters for template processing
type Parameters struct {
	context    string
	parameters map[string]string
}

func NewParameters(context string) *Parameters {
	return &Parameters{
		context:    context,
		parameters: make(map[string]string),
	}
}

func (p *Parameters) Get(key string) (string, bool) {
	value, exists := p.parameters[key]
	return value, exists
}

func (p *Parameters) SetFromMap(params map[string]string) {
	for key, value := range params {
		if _, exists := p.parameters[key]; exists {
			// If the key already exists, we log a warning, in yellow console color
			output.Warning("Overwriting existing parameter '%s' in context '%s'", key, p.context)
		}
		p.parameters[key] = value
	}
}

func (p *Parameters) Set(key, value string) {
	p.parameters[key] = value
}

func (p *Parameters) Merge(other *Parameters) *Parameters {
	if other == nil && p == nil {
		return nil
	}
	newParams := NewParameters("merged")
	if p != nil {
		newParams.SetFromMap(p.parameters)
		newParams.context = newParams.context + "-" + p.context
	}
	if other != nil {
		newParams.SetFromMap(other.parameters)
		newParams.context = newParams.context + "-" + other.context
	}
	return newParams
}
