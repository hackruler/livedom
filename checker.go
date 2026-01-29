package main

import (
	"crypto/tls"
	"fmt"
	"net"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/valyala/fasthttp"
)

// isPortOpen quickly checks if a TCP port is open (500ms timeout)
// This makes closed ports fail fast instead of waiting for HTTP timeout
func isPortOpen(host, port string) bool {
	timeout := 500 * time.Millisecond // Reduced to 500ms for even faster failures
	conn, err := net.DialTimeout("tcp", net.JoinHostPort(host, port), timeout)
	if err != nil {
		return false
	}
	conn.Close()
	return true
}

// checkSubdomain checks if a subdomain/URL is live and extracts relevant information
// Calls the callback function immediately for each result as it's ready (streaming)
func checkSubdomain(subdomain string, config *Config, callback func(Result)) {

	// Check if input is already a full URL
	var urls []string
	if strings.HasPrefix(subdomain, "http://") || strings.HasPrefix(subdomain, "https://") {
		// It's already a full URL, use it directly
		parsedURL, err := url.Parse(subdomain)
		if err == nil && len(config.Ports) > 0 {
			// If custom ports specified, create URLs with those ports
			for _, port := range config.Ports {
				portURL := fmt.Sprintf("%s://%s:%s%s", parsedURL.Scheme, parsedURL.Hostname(), port, parsedURL.Path)
				if parsedURL.RawQuery != "" {
					portURL += "?" + parsedURL.RawQuery
				}
				urls = append(urls, portURL)
			}
		} else {
			urls = []string{subdomain}
		}
	} else {
		// It's just a domain
		if len(config.Ports) > 0 {
			// Use custom ports - try both HTTP and HTTPS for each port
			for _, port := range config.Ports {
				// For standard ports, use appropriate scheme
				if port == "80" {
					urls = append(urls, fmt.Sprintf("http://%s:%s", subdomain, port))
				} else if port == "443" {
					urls = append(urls, fmt.Sprintf("https://%s:%s", subdomain, port))
				} else {
					// For non-standard ports, try both HTTP and HTTPS
					urls = append(urls, fmt.Sprintf("https://%s:%s", subdomain, port))
					urls = append(urls, fmt.Sprintf("http://%s:%s", subdomain, port))
				}
			}
		} else {
			// Default: try HTTPS first, then HTTP
			urls = []string{
				fmt.Sprintf("https://%s", subdomain),
				fmt.Sprintf("http://%s", subdomain),
			}
		}
	}

	// Create fasthttp client with optimized settings
	// Configure TLS to skip certificate validation (like httpx)
	tlsConfig := &tls.Config{
		InsecureSkipVerify: true,
	}
	
	client := &fasthttp.Client{
		MaxConnsPerHost:               200,
		MaxIdleConnDuration:           30 * time.Second,
		ReadTimeout:                   config.Timeout,
		WriteTimeout:                  config.Timeout,
		MaxIdemponentCallAttempts:     1,
		DisableHeaderNamesNormalizing: true,
		DisablePathNormalizing:        true,
		TLSConfig:                     tlsConfig,
	}

	// Check all URLs/ports in parallel using goroutines
	var wg sync.WaitGroup
	for _, targetURL := range urls {
		wg.Add(1)
		go func(targetURL string) {
			defer wg.Done()
			result := Result{
				URL:    targetURL,
				Input:  subdomain,
				Method: "GET",
				Failed: true, // Will be set to false on success
			}

			// Parse URL to get host and port for quick port check
			parsedURL, err := url.Parse(targetURL)
			if err != nil {
				return
			}
			
			host := parsedURL.Hostname()
			port := parsedURL.Port()
			if port == "" {
				if parsedURL.Scheme == "https" {
					port = "443"
				} else {
					port = "80"
				}
			}

			// Quick port check first (1 second timeout)
			// This makes closed ports fail fast instead of waiting for HTTP timeout
			if !isPortOpen(host, port) {
				return // Port is closed, exit immediately
			}

			// Port is open, proceed with HTTP request
			req := fasthttp.AcquireRequest()
			resp := fasthttp.AcquireResponse()
			defer fasthttp.ReleaseRequest(req)
			defer fasthttp.ReleaseResponse(resp)

			req.SetRequestURI(targetURL)
			req.Header.SetMethod("GET")
			req.Header.Set("User-Agent", "Mozilla/5.0")

			// Track request start time
			startTime := time.Now()

			// Use DoTimeout with shorter timeout for faster scanning
			requestTimeout := config.Timeout
			if requestTimeout > 5*time.Second {
				requestTimeout = 5 * time.Second // Cap at 5s for faster scanning
			}

			// Retry logic
			maxRetries := config.Retries
			if maxRetries < 0 {
				maxRetries = 0
			}
			
			for attempt := 0; attempt <= maxRetries; attempt++ {
				err = client.DoTimeout(req, resp, requestTimeout) // Use DoTimeout instead of Do
				if err == nil {
					break // Success
				}
				
				// If this was the last attempt, skip to cleanup
				if attempt >= maxRetries {
					break
				}
				
				// Wait before retry (reduced backoff delay)
				time.Sleep(time.Duration(attempt+1) * 50 * time.Millisecond) // Reduced from 100ms
			}
			
			// Release request/response immediately after use
			if err != nil {
				return // Exit goroutine if this URL failed
			}

		// Calculate request duration
		duration := time.Since(startTime)
		result.Time = duration.String()

		statusCode := resp.StatusCode()

		// Accept any response (including 4xx, 5xx) as "live"
		// This matches httpx behavior
		result.StatusCode = statusCode
		result.URL = targetURL
		result.Failed = false

		// Parse URL components (already parsed above, reuse)
		if parsedURL != nil {
			result.Scheme = parsedURL.Scheme
			result.Path = parsedURL.Path
			if result.Path == "" {
				result.Path = "/"
			}
			result.Port = port // Use port from earlier parsing
		}

		// Extract domain from URL for DNS resolution
		domain := extractDomainFromURL(targetURL)

		// Get headers
		result.ContentType = string(resp.Header.Peek("Content-Type"))
		result.Server = string(resp.Header.Peek("Server"))
		result.Location = string(resp.Header.Peek("Location"))

		// Collect all headers for tech detection
		headers := make(map[string]string)
		wappalyzerHeaders := make(map[string][]string)
		resp.Header.VisitAll(func(key, value []byte) {
			keyStr := string(key)
			valueStr := string(value)
			headers[keyStr] = valueStr
			// Also collect for Wappalyzer (needs []string format)
			wappalyzerHeaders[strings.ToLower(keyStr)] = []string{valueStr}
		})

		// Get content length from header, or use body length as fallback
		contentLength := resp.Header.ContentLength()
		if contentLength > 0 {
			result.ContentLength = int64(contentLength)
		} else {
			// If Content-Length header is not present, use actual body size
			body := resp.Body()
			result.ContentLength = int64(len(body))
		}

		// Read response body
		body := resp.Body()
		bodyStr := string(body)
		
		// For analysis (hash, title, tech detection), limit to 8KB for performance
		maxBodySize := 8192
		bodyForAnalysis := body
		if len(body) > maxBodySize {
			bodyForAnalysis = body[:maxBodySize]
		}
		bodyStrForAnalysis := string(bodyForAnalysis)

		// Always extract hash and title if JSON mode or flags are set
		if config.JSON || config.ShowHash || config.ShowTitle {
			if config.JSON || config.ShowHash {
				// If hash type is md5, calculate both body_md5 and header_md5
				hashType := config.HashType
				if hashType == "" {
					hashType = "sha256" // Default
				}
				
				// Only calculate hash when hash flag is explicitly set
				if config.ShowHash && hashType == "md5" {
					// Calculate body MD5 (use full body, not limited)
					result.Hash.BodyMD5 = calculateHash(body, "md5")
					
					// Calculate header MD5 - concatenate all headers
					var headerBytes []byte
					resp.Header.VisitAll(func(key, value []byte) {
						headerBytes = append(headerBytes, key...)
						headerBytes = append(headerBytes, ':')
						headerBytes = append(headerBytes, ' ')
						headerBytes = append(headerBytes, value...)
						headerBytes = append(headerBytes, '\r')
						headerBytes = append(headerBytes, '\n')
					})
					result.Hash.HeaderMD5 = calculateHash(headerBytes, "md5")
				}
			}
			if config.JSON || config.ShowTitle {
				title, _ := extractTitle(strings.NewReader(bodyStrForAnalysis))
				result.Title = title
			}
		}

		// Get favicon hash if requested
		if config.ShowFavicon {
			faviconHash := getFaviconHash(targetURL, client, config.Timeout)
			if faviconHash != "" {
				result.FaviconHash = faviconHash
			}
		}

		// JSON mode: collect all additional data
		if config.JSON {
			// Set timestamp
			result.Timestamp = time.Now().Format(time.RFC3339Nano)

			// Resolve all IPs
			if domain != "" {
				result.IPs = resolveAllIPs(domain)
				if len(result.IPs) > 0 {
					result.Host = result.IPs[0]
					result.IP = result.IPs[0] // Keep for backward compatibility

					// Detect CDN
					cdnName, cdnType, isCDN := detectCDN(result.IPs[0], headers)
					if isCDN {
						result.CDNName = cdnName
						result.CDNType = cdnType
						result.CDN = true
					}
				}

				// Resolve CNAME
				_, cname := resolveDNS(domain)
				result.CNAME = cname
			}

			// Get resolvers (simplified - get system DNS resolvers)
			result.Resolvers = getResolvers()

			// Count words and lines (matching httpx behavior)
			// Words: split by whitespace and count
			words := strings.Fields(bodyStr)
			result.Words = len(words)
			// Lines: split by newline (including empty lines)
			lines := strings.Split(bodyStr, "\n")
			// Remove trailing empty line if present (common in HTTP responses)
			if len(lines) > 0 && lines[len(lines)-1] == "" {
				result.Lines = len(lines) - 1
			} else {
				result.Lines = len(lines)
			}

			// Detect technologies using Wappalyzer
			result.Tech = detectTechnologies(targetURL, wappalyzerHeaders, bodyForAnalysis)

			// Knowledgebase (always set)
			pageType := getPageType(result.ContentType, bodyStrForAnalysis, result.StatusCode)
			if pageType == "" {
				pageType = "other"
			}
			// For pHash, use full body but limit calculation size
			pHash := calculatePHash(bodyStr, result.StatusCode)
			result.Knowledgebase = Knowledgebase{
				PageType: pageType,
				PHash:    pHash,
			}
		} else {
			// Resolve IP and CNAME if needed (non-JSON mode)
			if (config.ShowIP || config.ShowCNAME) && domain != "" {
				ip, cname := resolveDNS(domain)
				if config.ShowIP {
					result.IP = ip
				}
				if config.ShowCNAME {
					result.CNAME = cname
				}
			}
		}

		// Call callback immediately with this result (streaming output)
		callback(result)
		}(targetURL)
	}
	
	// Wait for all goroutines to complete
	wg.Wait()
}

// extractDomainFromURL extracts the domain from a URL string
func extractDomainFromURL(targetURL string) string {
	parsedURL, err := url.Parse(targetURL)
	var domain string
	if err == nil && parsedURL != nil {
		domain = parsedURL.Hostname()
	} else {
		// If URL parsing fails, try to extract domain from the URL string directly
		// Remove protocol if present
		domain = strings.TrimPrefix(strings.TrimPrefix(targetURL, "https://"), "http://")
		// Remove path, query, fragment
		if idx := strings.IndexAny(domain, "/?#"); idx != -1 {
			domain = domain[:idx]
		}
		// Remove port if present
		if idx := strings.Index(domain, ":"); idx != -1 {
			domain = domain[:idx]
		}
	}
	return domain
}
