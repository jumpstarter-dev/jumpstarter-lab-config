/*
Copyright 2025. The Jumpstarter Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package main

import (
	"bytes"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/charmbracelet/glamour"
	"github.com/mattn/go-runewidth"
	"github.com/spf13/cobra"

	"github.com/jumpstarter-dev/jumpstarter-lab-config/api/v1alpha1"
	"github.com/jumpstarter-dev/jumpstarter-lab-config/internal/config"
	"github.com/jumpstarter-dev/jumpstarter-lab-config/internal/templating"
	"github.com/jumpstarter-dev/jumpstarter-lab-config/internal/vars"
)

var docsCmd = &cobra.Command{
	Use:   "docs [config-file]",
	Short: "Generate documentation for configured DUTs",
	Long: `Generate markdown documentation table for all Device Under Test (DUT) boards with location and ` +
		`status information. You do not need vault password file when all the fields in the output are not encrypted.`,
	Args:         cobra.MaximumNArgs(1),
	SilenceUsage: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		vaultPassFile, _ := cmd.Flags().GetString("vault-password-file")
		outFile, _ := cmd.Flags().GetString("out")

		configFilePath := defaultConfigFile
		if len(args) > 0 {
			configFilePath = args[0]
		}

		cfg, err := config.LoadConfig(configFilePath, vaultPassFile)
		if err != nil {
			return fmt.Errorf("error loading config file %s: %w", configFilePath, err)
		}

		docsCfg := cfg
		if vaultPassFile == "" {
			if docsCfg, err = placeholderEncryptedVars(cfg); err != nil {
				return fmt.Errorf("error creating docs config: %w", err)
			}
		}

		// Generate DUT documentation
		err = generateDUTDocumentation(docsCfg, outFile)
		if err != nil {
			return fmt.Errorf("error generating documentation: %w", err)
		}

		return nil
	},
}

// DUTInfo represents the information we want to display for each DUT
type DUTInfo struct {
	Name         string
	LocationName string
	Location     string
	Notes        string
}

// LocationGroup represents a group of DUTs by location
type LocationGroup struct {
	Name        string
	Description string
	DUTs        []DUTInfo
}

func placeholderEncryptedVars(cfg *config.Config) (*config.Config, error) {
	// Create a mock variables instance that provides placeholder values for all variables
	mockVars, err := vars.NewVariables("")
	if err != nil {
		return nil, fmt.Errorf("error creating mock variables: %w", err)
	}

	// Add placeholder values for all variables that might be vault-encrypted
	if cfg.Loaded.Variables != nil {
		for _, varKey := range cfg.Loaded.Variables.GetAllKeys() {
			if cfg.Loaded.Variables.Has(varKey) {
				// Try to get the actual value, but fall back to placeholder if it fails
				actualValue, err := cfg.Loaded.Variables.Get(varKey)
				if err != nil {
					// This is likely a vault-encrypted variable, use a placeholder
					if setErr := mockVars.Set(varKey, "[REDACTED]"); setErr != nil {
						return nil, fmt.Errorf("error setting mock variable %s: %w", varKey, setErr)
					}
				} else {
					if setErr := mockVars.Set(varKey, actualValue); setErr != nil {
						return nil, fmt.Errorf("error setting variable %s: %w", varKey, setErr)
					}
				}
			}
		}
	}

	// Create a copy of the loaded config with mock variables
	docsCfg := &config.Config{
		Sources:   cfg.Sources,
		Variables: cfg.Variables,
		BaseDir:   cfg.BaseDir,
		Loaded: &config.LoadedLabConfig{
			Clients:                 cfg.Loaded.Clients,
			Policies:                cfg.Loaded.Policies,
			PhysicalLocations:       cfg.Loaded.PhysicalLocations,
			ExporterHosts:           cfg.Loaded.ExporterHosts,
			ExporterInstances:       cfg.Loaded.ExporterInstances,
			ExporterConfigTemplates: cfg.Loaded.ExporterConfigTemplates,
			JumpstarterInstances:    cfg.Loaded.JumpstarterInstances,
			Variables:               mockVars, // Use mock variables instead
			SourceFiles:             cfg.Loaded.SourceFiles,
		},
	}

	return docsCfg, nil
}

func generateDUTDocumentation(cfg *config.Config, outFile string) error {
	// Generate the markdown content
	markdownContent := generateMarkdownContent(cfg)

	// Save to file if outFile is specified
	if outFile != "" {
		err := os.WriteFile(outFile, []byte(markdownContent), 0644)
		if err != nil {
			return fmt.Errorf("error writing to file %s: %w", outFile, err)
		}
		fmt.Printf("Documentation written to %s\n", outFile)
		// When writing to file, don't render to terminal
		return nil
	}

	// Only render to terminal when no output file is specified
	return renderToTerminal(markdownContent)
}

func generateMarkdownContent(cfg *config.Config) string {
	var buf bytes.Buffer
	// Create a map of physical locations for enriched location info
	locationMap := make(map[string]*v1alpha1.PhysicalLocation)
	for _, location := range cfg.Loaded.PhysicalLocations {
		locationCopy := location.DeepCopy()
		locationMap[locationCopy.Name] = locationCopy
	}

	// Apply templates to export instances safely with mock variables
	processedExporters := make([]*v1alpha1.ExporterInstance, 0, len(cfg.Loaded.ExporterInstances))
	tapplier, err := templating.NewTemplateApplier(cfg, nil)
	if err != nil {
		fmt.Printf("Warning: Could not create template applier: %v\n", err)
		// Fall back to using unprocessed exporters
		for _, exp := range cfg.Loaded.ExporterInstances {
			processedExporters = append(processedExporters, exp)
		}
	} else {
		// Try to apply templates with mock variables
		for _, exporterInstance := range cfg.Loaded.ExporterInstances {
			exporterCopy := exporterInstance.DeepCopy()
			err := tapplier.Apply(exporterCopy)
			if err != nil {
				// If template application fails, use the original
				fmt.Printf("Warning: Template application failed for %s: %v\n", exporterInstance.Name, err)
				processedExporters = append(processedExporters, exporterInstance)
			} else {
				processedExporters = append(processedExporters, exporterCopy)
			}
		}
	}

	// Group DUTs by location
	locationGroups := make(map[string]*LocationGroup)
	for _, exporterInstance := range processedExporters {
		locationName := getLocationName(exporterInstance)

		// Create location group if it doesn't exist
		if _, exists := locationGroups[locationName]; !exists {
			description := ""
			if physLoc, exists := locationMap[locationName]; exists {
				description = physLoc.Spec.Description
			}
			locationGroups[locationName] = &LocationGroup{
				Name:        locationName,
				Description: description,
				DUTs:        make([]DUTInfo, 0),
			}
		}

		dut := DUTInfo{
			Name:         exporterInstance.Name,
			LocationName: locationName,
			Location:     formatLocation(exporterInstance),
			Notes:        formatNotes(exporterInstance),
		}
		locationGroups[locationName].DUTs = append(locationGroups[locationName].DUTs, dut)
	}

	// Sort location groups by name
	sortedLocationNames := make([]string, 0, len(locationGroups))
	for locationName := range locationGroups {
		sortedLocationNames = append(sortedLocationNames, locationName)
	}
	sort.Strings(sortedLocationNames)

	// Generate markdown table grouped by location
	buf.WriteString("# DUT Documentation\n\n")

	for _, locationName := range sortedLocationNames {
		group := locationGroups[locationName]

		// Sort DUTs within each location alphabetically
		sort.Slice(group.DUTs, func(i, j int) bool {
			return group.DUTs[i].Name < group.DUTs[j].Name
		})

		// Write location header
		buf.WriteString("## " + escapeMarkdown(group.Name))
		if group.Description != "" {
			buf.WriteString(" - " + escapeMarkdown(group.Description))
		}
		buf.WriteString("\n\n")

		// Write table for this location
		buf.WriteString("| Name | Location | Notes |\n")
		buf.WriteString("|------|----------|-------|\n")

		for _, dut := range group.DUTs {
			// Escape markdown special characters in all fields
			escapedName := escapeMarkdown(dut.Name)
			escapedLocation := escapeMarkdown(dut.Location)
			escapedNotes := escapeMarkdown(dut.Notes)
			buf.WriteString(fmt.Sprintf("| %s | %s | %s |\n", escapedName, escapedLocation, escapedNotes))
		}
		buf.WriteString("\n")
	}

	return buf.String()
}

func renderToTerminal(markdownContent string) error {
	// Parse markdown content and render with custom table formatting
	lines := strings.Split(markdownContent, "\n")
	inTable := false
	var tableRows [][]string
	var headers []string

	renderer, err := glamour.NewTermRenderer(glamour.WithAutoStyle())
	if err != nil {
		// Fall back to plain text if glamour fails
		renderer = nil
	}

	for _, line := range lines {
		line = strings.TrimSpace(line)

		// Check if this is a table row
		if strings.HasPrefix(line, "|") && strings.HasSuffix(line, "|") {
			// Parse table row
			cells := strings.Split(line, "|")
			// Remove first and last empty elements
			if len(cells) > 2 {
				cells = cells[1 : len(cells)-1]
			}
			// Trim spaces from each cell
			for i, cell := range cells {
				cells[i] = strings.TrimSpace(cell)
			}

			if !inTable {
				headers = cells
				inTable = true
			} else if !strings.Contains(line, "---") { // Skip separator line
				tableRows = append(tableRows, cells)
			}
		} else if inTable && line == "" {
			// End of table - render it with dynamic name column width
			renderBoxedTableWithDynamicNameWidth(headers, tableRows)
			inTable = false
			tableRows = nil
			headers = nil
			fmt.Println()
		} else if !inTable {
			// Render non-table content using Glamour
			if strings.HasPrefix(line, "#") && renderer != nil {
				rendered, err := renderer.Render(line + "\n")
				if err == nil {
					fmt.Print(rendered)
					continue
				}
			}
			fmt.Println(line)
		}
	}

	// Handle table at end of content
	if inTable && len(headers) > 0 {
		renderBoxedTableWithDynamicNameWidth(headers, tableRows)
	}

	return nil
}

func renderBoxedTableWithDynamicNameWidth(headers []string, rows [][]string) {
	if len(headers) == 0 {
		return
	}

	// Calculate dynamic name column width based on the longest name in this table
	maxNameWidth := runewidth.StringWidth(headers[0]) // Start with header width
	for _, row := range rows {
		if len(row) > 0 {
			nameWidth := runewidth.StringWidth(row[0])
			if nameWidth > maxNameWidth {
				maxNameWidth = nameWidth
			}
		}
	}

	// Set column widths: dynamic name width, fixed location and notes widths
	maxColWidths := []int{maxNameWidth, 15, 80} // Name (dynamic), Location, Notes
	colWidths := make([]int, len(headers))

	// Set the calculated widths
	for i, header := range headers {
		if i < len(maxColWidths) {
			colWidths[i] = maxColWidths[i]
			// Ensure header fits
			headerWidth := runewidth.StringWidth(header)
			if headerWidth > colWidths[i] {
				colWidths[i] = headerWidth
			}
		}
	}

	// Check content widths but respect maximums for location and notes columns
	for _, row := range rows {
		for i, cell := range row {
			if i < len(colWidths) && i < len(maxColWidths) {
				cellWidth := runewidth.StringWidth(cell)
				if i == 0 {
					// For name column, use the calculated dynamic width
					continue
				} else if cellWidth > colWidths[i] {
					if cellWidth <= maxColWidths[i] {
						colWidths[i] = cellWidth
					} else {
						colWidths[i] = maxColWidths[i]
					}
				}
			}
		}
	}

	// Add padding
	for i := range colWidths {
		colWidths[i] += 2 // 1 space on each side
	}

	renderTableWithWidths(headers, rows, colWidths)
}

func renderTableWithWidths(headers []string, rows [][]string, colWidths []int) {
	// Render top border
	fmt.Print("  ┌")
	for i, width := range colWidths {
		fmt.Print(strings.Repeat("─", width))
		if i < len(colWidths)-1 {
			fmt.Print("┬")
		}
	}
	fmt.Println("┐")

	// Render header row
	fmt.Print("  │")
	for i, header := range headers {
		fmt.Printf(" %-*s │", colWidths[i]-2, header)
	}
	fmt.Println()

	// Render header separator
	fmt.Print("  ├")
	for i, width := range colWidths {
		fmt.Print(strings.Repeat("─", width))
		if i < len(colWidths)-1 {
			fmt.Print("┼")
		}
	}
	fmt.Println("┤")

	// Render data rows with text wrapping
	for rowIdx, row := range rows {
		// Wrap text in each cell
		wrappedCells := make([][]string, len(row))
		maxLines := 1

		for i, cell := range row {
			if i < len(colWidths) {
				// Clean up the cell content (remove <br/> tags)
				cleanCell := strings.ReplaceAll(cell, "<br/>", " ")
				wrappedCells[i] = wrapText(cleanCell, colWidths[i]-2)
				if len(wrappedCells[i]) > maxLines {
					maxLines = len(wrappedCells[i])
				}
			}
		}

		// Render each line of the wrapped cell content
		for lineIdx := 0; lineIdx < maxLines; lineIdx++ {
			fmt.Print("  │")
			for i := range row {
				if i < len(colWidths) {
					var lineText string
					if lineIdx < len(wrappedCells[i]) {
						lineText = wrappedCells[i][lineIdx]
					}
					fmt.Printf(" %-*s │", colWidths[i]-2, lineText)
				}
			}
			fmt.Println()
		}

		// Add row separator (except for last row)
		if rowIdx < len(rows)-1 {
			fmt.Print("  ├")
			for i, width := range colWidths {
				fmt.Print(strings.Repeat("─", width))
				if i < len(colWidths)-1 {
					fmt.Print("┼")
				}
			}
			fmt.Println("┤")
		}
	}

	// Render bottom border
	fmt.Print("  └")
	for i, width := range colWidths {
		fmt.Print(strings.Repeat("─", width))
		if i < len(colWidths)-1 {
			fmt.Print("┴")
		}
	}
	fmt.Println("┘")
}

func wrapText(text string, width int) []string {
	if width <= 0 {
		return []string{text}
	}

	words := strings.Fields(text)
	if len(words) == 0 {
		return []string{""}
	}

	var lines []string
	var currentLine strings.Builder

	for _, word := range words {
		// If this is the first word in the line
		if currentLine.Len() == 0 {
			currentLine.WriteString(word)
		} else {
			// Check if adding this word would exceed the width
			testLine := currentLine.String() + " " + word
			if runewidth.StringWidth(testLine) <= width {
				currentLine.WriteString(" " + word)
			} else {
				// Start a new line
				lines = append(lines, currentLine.String())
				currentLine.Reset()
				currentLine.WriteString(word)
			}
		}
	}

	// Add the last line if it has content
	if currentLine.Len() > 0 {
		lines = append(lines, currentLine.String())
	}

	if len(lines) == 0 {
		return []string{""}
	}

	return lines
}

func getLocationName(exporterInstance *v1alpha1.ExporterInstance) string {
	// Use DutLocationRef.Name for grouping, otherwise group as "on-desk"
	if exporterInstance.Spec.DutLocationRef.Name != "" {
		return exporterInstance.Spec.DutLocationRef.Name
	}

	// All devices without DutLocationRef go to "on-desk" group
	return "on-desk"
}

func formatLocation(exporterInstance *v1alpha1.ExporterInstance) string {
	dutLoc := exporterInstance.Spec.DutLocationRef

	// DutLocationRef has precedence - format as "rack / tray" or just "rack"
	if dutLoc.Rack != "" && dutLoc.Tray != "" {
		return dutLoc.Rack + " / " + dutLoc.Tray
	} else if dutLoc.Rack != "" {
		return dutLoc.Rack
	}

	// Fall back to "location" label in spec.labels if DutLocationRef is not available
	if exporterInstance.Spec.Labels != nil {
		if locationLabel, exists := exporterInstance.Spec.Labels["location"]; exists && locationLabel != "" {
			return locationLabel
		}
	}

	return "N/A"
}

func formatNotes(exporterInstance *v1alpha1.ExporterInstance) string {
	var noteParts []string

	// Check for "dead" annotation and add to notes FIRST
	if deadAnnotation, exists := exporterInstance.Annotations["dead"]; exists {
		deadNote := "**DEAD**"
		if deadAnnotation != "" {
			deadNote += ": " + deadAnnotation
		}
		noteParts = append(noteParts, deadNote)
	}

	// Add spec notes if available
	if exporterInstance.Spec.Notes != "" {
		noteParts = append(noteParts, strings.TrimSpace(exporterInstance.Spec.Notes))
	}

	// Return empty string if no notes (don't show N/A)
	if len(noteParts) == 0 {
		return ""
	}

	return strings.Join(noteParts, " • ")
}

func escapeMarkdown(text string) string {
	// Replace newlines with <br/> for markdown table compatibility
	text = strings.ReplaceAll(text, "\n", "<br/>")
	// Escape backslashes first to avoid double-escaping
	text = strings.ReplaceAll(text, "\\", "\\\\")
	// Escape pipe characters that would break the table
	text = strings.ReplaceAll(text, "|", "\\|")
	return text
}

func init() {
	// Add the vault password file flag
	docsCmd.Flags().String("vault-password-file", "", "Path to the vault password file for decrypting variables")
	// Add the output file flag
	docsCmd.Flags().String("out", "", "Output file path for markdown documentation (optional)")
	// Add the docs command to the root command
	rootCmd.AddCommand(docsCmd)
}
