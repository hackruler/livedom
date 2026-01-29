package main

import (
	"github.com/fatih/color"
)

// newColor creates a new color with colors always enabled
func newColor(attr color.Attribute) *color.Color {
	c := color.New(attr)
	c.EnableColor()
	return c
}

// truncateString truncates a string to a maximum length, adding "..." if needed
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}

// getStatusColor returns a color function based on the HTTP status code
func getStatusColor(statusCode int) func(a ...interface{}) string {
	switch {
	case statusCode >= 200 && statusCode < 300:
		// 2xx - Green (success)
		return color.New(color.FgGreen).SprintFunc()
	case statusCode >= 300 && statusCode < 400:
		// 3xx - Yellow (redirect)
		return color.New(color.FgYellow).SprintFunc()
	case statusCode >= 400 && statusCode < 500:
		// 4xx - Red (client error)
		return color.New(color.FgRed).SprintFunc()
	case statusCode >= 500:
		// 5xx - Magenta (server error)
		return color.New(color.FgMagenta).SprintFunc()
	default:
		// Other - White
		return color.New(color.FgWhite).SprintFunc()
	}
}
