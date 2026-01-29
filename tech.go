package main

import (
	wappalyzer "github.com/projectdiscovery/wappalyzergo"
)

// detectTechnologies detects technologies using Wappalyzer
func detectTechnologies(url string, headers map[string][]string, body []byte) []string {
	// Create Wappalyzer instance
	wappalyzerClient, err := wappalyzer.New()
	if err != nil {
		// If Wappalyzer fails to initialize, return empty slice
		return []string{}
	}

	// Run Wappalyzer fingerprinting
	// Headers are already in the correct format (map[string][]string)
	fingerprints := wappalyzerClient.Fingerprint(headers, body)

	// Extract technology names
	var tech []string
	for app := range fingerprints {
		tech = append(tech, app)
	}

	return tech
}
