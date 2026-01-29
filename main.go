package main

import (
	"os"

	"github.com/fatih/color"
)

func main() {
	// Force color output even when redirecting to file
	// This ensures ANSI color codes are written to files (like httpx)
	color.NoColor = false
	// Override the output to always enable colors
	color.Output = os.Stdout

	config := parseFlags()

	// Check if update flag is set
	if config.Update {
		updateTool()
		return
	}

	// Check if single URL flag is set
	if config.URL != "" {
		checkSubdomain(config.URL, config, func(result Result) {
			if result.Error == nil {
				displaySingleResult(result, config)
			}
		})
		return
	}

	// Process subdomains as they come in (streaming)
	processSubdomainsStreaming(config)
}
