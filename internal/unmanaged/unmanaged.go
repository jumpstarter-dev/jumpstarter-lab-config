package unmanaged

import (
	"fmt"
	"math"
	"os"
	"time"

	api "github.com/jumpstarter-dev/jumpstarter-lab-config/api/v1alpha1"
	"github.com/jumpstarter-dev/jumpstarter-lab-config/internal/config"
)

type Summary struct {
	Exporters  map[string]bool
	OldestDays int
}

func (s Summary) Count() int {
	return len(s.Exporters)
}

// ProcessUnmanagedExporters iterates over ExporterInstances and processes the
// unmanaged annotation. For each unmanaged exporter:
//   - If the annotation value is empty, sets today's date and writes it back to the source file.
//   - If the annotation value is a date, calculates days since and prints a warning.
//   - If the annotation value is invalid, warns but still treats as unmanaged.
//
// Returns a summary with unmanaged exporter names and oldest age.
// nowFunc is injected to make time-based behavior deterministic in tests.
func ProcessUnmanagedExporters(cfg *config.Config, dryRun bool, nowFunc func() time.Time) Summary {
	summary := Summary{
		Exporters: make(map[string]bool),
	}
	if nowFunc == nil {
		nowFunc = time.Now
	}

	if cfg == nil || cfg.Loaded == nil {
		return summary
	}

	for _, exporterInstance := range cfg.Loaded.ExporterInstances {
		if exporterInstance == nil {
			continue
		}

		isUnmanaged, value := exporterInstance.IsUnmanaged()
		if !isUnmanaged {
			continue
		}

		summary.Exporters[exporterInstance.Name] = true
		daysUnmanaged := 0

		if value == "" {
			// First discovery - set today's date
			today := nowFunc().Format("2006-01-02")
			_, _ = fmt.Fprintf(os.Stderr, "⚠️  Warning: exporter %s is unmanaged and missing discovery date\n", exporterInstance.Name)

			if dryRun {
				_, _ = fmt.Fprintf(os.Stderr, "⚠️  Warning: dry run - would set %s=%s for exporter %s\n",
					api.UnmanagedAnnotation, today, exporterInstance.Name)
			} else {
				// Write back to source file
				sourceFile := getSourceFile(cfg, exporterInstance.Name)
				if sourceFile != "" {
					if err := config.UpdateAnnotationInSourceFile(sourceFile, exporterInstance.Name, api.UnmanagedAnnotation, today); err != nil {
						_, _ = fmt.Fprintf(os.Stderr, "⚠️  Warning: failed to write discovery date to %s: %v\n", sourceFile, err)
					} else {
						// Keep in-memory state consistent with source file.
						if exporterInstance.Annotations == nil {
							exporterInstance.Annotations = make(map[string]string)
						}
						exporterInstance.Annotations[api.UnmanagedAnnotation] = today
					}
				} else {
					_, _ = fmt.Fprintf(os.Stderr, "⚠️  Warning: source file not found for unmanaged exporter %s\n", exporterInstance.Name)
				}
			}
		} else {
			// Parse the date and calculate days since
			discoveryDate, err := time.Parse("2006-01-02", value)
			if err != nil {
				_, _ = fmt.Fprintf(os.Stderr, "⚠️  Warning: exporter %s is unmanaged (invalid date: %s)\n", exporterInstance.Name, value)
			} else {
				now := nowFunc()
				if discoveryDate.After(now) {
					_, _ = fmt.Fprintf(
						os.Stderr,
						"⚠️  Warning: exporter %s has unmanaged discovery date in the future: %s\n",
						exporterInstance.Name,
						value,
					)
					daysUnmanaged = 0
				} else {
					daysUnmanaged = int(math.Floor(now.Sub(discoveryDate).Hours() / 24))
					_, _ = fmt.Fprintf(os.Stderr, "⚠️  Warning: exporter %s has been unmanaged for %d days\n", exporterInstance.Name, daysUnmanaged)
				}
			}
		}

		if daysUnmanaged > summary.OldestDays {
			summary.OldestDays = daysUnmanaged
		}
	}

	return summary
}

// getSourceFile looks up the source file for an ExporterInstance
func getSourceFile(cfg *config.Config, name string) string {
	if cfg.Loaded.SourceFiles == nil {
		return ""
	}
	exporterSources, ok := cfg.Loaded.SourceFiles["ExporterInstance"]
	if !ok {
		return ""
	}
	return exporterSources[name]
}
