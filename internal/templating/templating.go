package templating

import (
	"fmt"
	"regexp"

	"github.com/jumpstarter-dev/jumpstarter-lab-config/internal/vars"
)

func ProcessTemplate(data string, variables *vars.Variables, parameters *Parameters) (string, error) {
	// This function would process the template using the provided variables.
	// For now, we will just return the template as-is for demonstration purposes.
	// In a real implementation, you would use a templating engine like text/template or html/template.
	if needsReplacements(data) {
		replacements, err := constructReplacementMap(variables, parameters)
		if err != nil {
			return "", err
		}
		return applyReplacements(data, replacements)
	}

	return data, nil
}

func needsReplacements(data string) bool {
	// if "$(.*)" is found anywhere in the data, it indicates that replacements are needed
	// check with a regex
	return regexp.MustCompile(`\$\((.*?)\)`).MatchString(data)

}

func constructReplacementMap(variables *vars.Variables, parameters *Parameters) (map[string]string, error) {
	replacements := make(map[string]string)

	// Add variables to the replacement map
	for _, key := range variables.GetAllKeys() {
		value, err := variables.Get(key)
		if err != nil {
			return nil, fmt.Errorf("templating: error retrieving variable '%s': %w", key, err)
		}
		replacements["var."+key] = value
	}

	// Add parameters to the replacement map
	for key, value := range parameters.parameters {
		replacements["param."+key] = value
	}

	return replacements, nil
}

func applyReplacements(data string, replacements map[string]string) (string, error) {
	const RECURSION_LIMIT = 10 // Limit iterations to prevent infinite loops, i.e. var.a = $(var.a) or similar
	var (
		i                               int
		replacementsContainReplacements bool
		recursiveReplacementInfo        string
	)

	for i = 0; i < RECURSION_LIMIT; i++ {
		// DEBUG: fmt.Printf("Applying replacements, iteration %d, input: %s\n", i, data)
		replacementsContainReplacements = false
		// Apply replacements to the data
		for key, value := range replacements {
			keyRegexp := regexp.MustCompile(`\$\(\s*` + key + `\s*\)`)
			hasKeyReplacement := keyRegexp.MatchString(data)
			data = keyRegexp.ReplaceAllString(data, value)
			// if the key was found in the data, check if the value contains any replacements
			if hasKeyReplacement && needsReplacements(value) {
				// in such case, our replacement contains new replacements and we will need to iterate again
				replacementsContainReplacements = true
				recursiveReplacementInfo = fmt.Sprintf("%s => %s", key, value)
			}
		}
		if !replacementsContainReplacements {
			break // No more replacements needed
		}
	}
	// DEBUG: fmt.Printf("Finished replacements, iteration %d, output: %s\n", i, data)
	if i == RECURSION_LIMIT && replacementsContainReplacements {
		return data, fmt.Errorf("templating: recursion limit reached while applying replacements, "+
			"check for circular references, like: %s", recursiveReplacementInfo)
	}
	// find unhandled variables and return an error for the ones not found
	unhandled := regexp.MustCompile(`\$\(\s*(.*?)\s*\)`)
	matches := unhandled.FindAllStringSubmatch(data, -1)
	if len(matches) > 0 {
		var missingKeys []string
		for _, match := range matches {
			if len(match) > 1 {
				missingKeys = append(missingKeys, match[1])
			}
		}
		if len(missingKeys) > 0 {
			return data, fmt.Errorf("templating: unhandled variables found: %v", missingKeys)
		}
	}
	// If no unhandled variables are found, return the modified data
	return data, nil
}
