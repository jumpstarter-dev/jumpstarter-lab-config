package output

import "fmt"

func Warning(format string, args ...interface{}) {
	// Print a warning message to the console in yellow color

	// Example using fmt.Printf with ANSI escape codes for yellow text
	fmt.Printf("\033[33mWarning: "+format+"\033[0m\n", args...)
}
