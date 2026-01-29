package main

import (
	"net"
	"strings"
)

// detectCDN detects CDN based on IP address and headers
func detectCDN(ip string, headers map[string]string) (string, string, bool) {
	// Check IP ranges for known CDNs
	if isGoogleCDN(ip) {
		return "google", "cdn", true
	}
	if isCloudflareCDN(ip) {
		return "cloudflare", "cdn", true
	}
	if isAWSCloudFront(ip) {
		return "cloudfront", "cdn", true
	}
	if isFastlyCDN(ip) {
		return "fastly", "cdn", true
	}

	// Check headers for CDN indicators
	server := strings.ToLower(headers["Server"])
	via := strings.ToLower(headers["Via"])
	cfRay := headers["CF-Ray"]
	
	if cfRay != "" {
		return "cloudflare", "cdn", true
	}
	if strings.Contains(server, "cloudflare") {
		return "cloudflare", "cdn", true
	}
	if strings.Contains(via, "cloudflare") {
		return "cloudflare", "cdn", true
	}
	if strings.Contains(server, "cloudfront") {
		return "cloudfront", "cdn", true
	}
	if strings.Contains(server, "fastly") {
		return "fastly", "cdn", true
	}

	return "", "", false
}

// isGoogleCDN checks if IP belongs to Google CDN
func isGoogleCDN(ip string) bool {
	parsedIP := net.ParseIP(ip)
	if parsedIP == nil {
		return false
	}

	// Google Cloud CDN IP ranges (simplified check)
	// In production, you'd want to check against actual CIDR ranges
	if parsedIP.To4() != nil {
		ip4 := parsedIP.To4()
		// Check for common Google Cloud IP ranges
		// 34.x.x.x, 35.x.x.x, 104.x.x.x, 130.x.x.x, etc.
		firstOctet := ip4[0]
		if firstOctet == 34 || firstOctet == 35 || firstOctet == 104 || firstOctet == 130 {
			return true
		}
	}

	return false
}

// isCloudflareCDN checks if IP belongs to Cloudflare CDN
func isCloudflareCDN(ip string) bool {
	parsedIP := net.ParseIP(ip)
	if parsedIP == nil {
		return false
	}

	// Cloudflare IP ranges (simplified)
	if parsedIP.To4() != nil {
		ip4 := parsedIP.To4()
		// Cloudflare uses 104.x.x.x, 172.x.x.x, 198.x.x.x, etc.
		firstOctet := ip4[0]
		if firstOctet == 104 || firstOctet == 172 || firstOctet == 198 {
			return true
		}
	}

	return false
}

// isAWSCloudFront checks if IP belongs to AWS CloudFront
func isAWSCloudFront(ip string) bool {
	parsedIP := net.ParseIP(ip)
	if parsedIP == nil {
		return false
	}

	// AWS CloudFront IP ranges (simplified)
	if parsedIP.To4() != nil {
		ip4 := parsedIP.To4()
		// CloudFront uses various ranges
		firstOctet := ip4[0]
		if firstOctet == 13 || firstOctet == 52 || firstOctet == 54 {
			return true
		}
	}

	return false
}

// isFastlyCDN checks if IP belongs to Fastly CDN
func isFastlyCDN(ip string) bool {
	parsedIP := net.ParseIP(ip)
	if parsedIP == nil {
		return false
	}

	// Fastly IP ranges (simplified)
	if parsedIP.To4() != nil {
		ip4 := parsedIP.To4()
		// Fastly uses 151.x.x.x, etc.
		firstOctet := ip4[0]
		if firstOctet == 151 {
			return true
		}
	}

	return false
}
