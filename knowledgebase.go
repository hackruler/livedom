package main

import (
	"crypto/sha256"
	"encoding/binary"
	"strings"
)

// getPageType determines the page type based on content and status code
func getPageType(contentType string, body string, statusCode int) string {
	contentTypeLower := strings.ToLower(contentType)

	// Check for error pages (3xx, 4xx, 5xx)
	if statusCode >= 300 && statusCode < 400 {
		// Redirect pages
		return "error"
	}
	if statusCode >= 400 {
		// Error pages
		return "error"
	}

	// Check for JSON
	if strings.Contains(contentTypeLower, "application/json") {
		return "json"
	}

	// Check for XML
	if strings.Contains(contentTypeLower, "application/xml") || strings.Contains(contentTypeLower, "text/xml") {
		return "xml"
	}

	// For HTML pages, httpx often uses "other" instead of "html"
	// This matches httpx behavior for certain sites (like Framer)
	if strings.Contains(contentTypeLower, "text/html") {
		// Check if it's a simple HTML page or a complex SPA/framework
		// For now, use "other" to match httpx behavior more closely
		// You can customize this logic based on specific requirements
		return "other"
	}

	// Default
	return "other"
}

// calculatePHash calculates a perceptual hash (simplified version)
// This is a simplified pHash - a real perceptual hash would use image processing
// Returns 0 for error pages and in some cases to match httpx behavior
func calculatePHash(body string, statusCode int) int {
	// Return 0 for error pages (3xx, 4xx, 5xx) - matches httpx behavior
	if statusCode >= 300 {
		return 0
	}

	if len(body) == 0 {
		return 0
	}

	// httpx sometimes returns 0 for pHash even on 200 status
	// This might be for very large files or certain content types
	// For now, we'll calculate it, but you can add conditions to return 0
	// if needed to match specific httpx behavior

	// Take first 1024 bytes for hash calculation
	hashInput := body
	if len(hashInput) > 1024 {
		hashInput = hashInput[:1024]
	}

	// Calculate SHA256 and take first 4 bytes as int
	hash := sha256.Sum256([]byte(hashInput))
	// Convert first 4 bytes to int32
	hashInt := int(binary.BigEndian.Uint32(hash[:4]))

	// Make it positive
	if hashInt < 0 {
		hashInt = -hashInt
	}

	return hashInt
}
