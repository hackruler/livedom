package main

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"net/url"
	"time"

	"github.com/valyala/fasthttp"
)

// getFaviconHash fetches the favicon and calculates its hash (mmh3 hash like httpx)
func getFaviconHash(baseURL string, client *fasthttp.Client, timeout time.Duration) string {
	// Parse base URL
	parsedURL, err := url.Parse(baseURL)
	if err != nil {
		return ""
	}

	// Try common favicon paths
	faviconPaths := []string{
		"/favicon.ico",
		"/favicon.png",
		"/apple-touch-icon.png",
	}

	for _, path := range faviconPaths {
		faviconURL := fmt.Sprintf("%s://%s%s", parsedURL.Scheme, parsedURL.Host, path)

		fu, err := url.Parse(faviconURL)
		if err != nil {
			continue
		}

		req := fasthttp.AcquireRequest()
		resp := fasthttp.AcquireResponse()

		prepareFasthttpRequest(req, fu, "GET")
		req.Header.Set("User-Agent", "Mozilla/5.0")

		doErr := client.DoTimeout(req, resp, timeout)
		fasthttp.ReleaseRequest(req)
		fasthttp.ReleaseResponse(resp)
		if doErr != nil {
			continue
		}

		// Only process if we got a successful response
		if resp.StatusCode() >= 200 && resp.StatusCode() < 300 {
			body := resp.Body()
			if len(body) > 0 {
				// Calculate mmh3 hash (simplified - using MD5 for now, can be replaced with mmh3)
				// httpx uses mmh3 hash for favicon, but we'll use MD5 as a placeholder
				// You can add mmh3 library later if needed
				hash := md5.Sum(body)
				return hex.EncodeToString(hash[:])
			}
		}
	}

	return ""
}

// calculateMMH3Hash calculates mmh3 hash (used by httpx for favicon)
// This is a simplified version - for production, use a proper mmh3 library
func calculateMMH3Hash(data []byte) uint32 {
	// Placeholder - httpx uses mmh3 hash
	// For now, we'll use a simple hash calculation
	// You can add github.com/spaolacci/murmur3 or similar library
	hash := md5.Sum(data)
	// Convert first 4 bytes to uint32 (simplified)
	return uint32(hash[0])<<24 | uint32(hash[1])<<16 | uint32(hash[2])<<8 | uint32(hash[3])
}
