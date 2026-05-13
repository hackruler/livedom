package main

import (
	"crypto/tls"
	"fmt"
	"net"
	"net/url"
	"strings"
	"time"

	"github.com/valyala/fasthttp"
)

// prioritizeHTTPS returns a copy of urls with all https:// entries before http:// entries (stable within each group).
func prioritizeHTTPS(urls []string) []string {
	var https, other []string
	for _, u := range urls {
		if strings.HasPrefix(u, "https://") {
			https = append(https, u)
		} else if strings.HasPrefix(u, "http://") {
			other = append(other, u)
		} else {
			other = append(other, u)
		}
	}
	out := make([]string, 0, len(urls))
	out = append(out, https...)
	out = append(out, other...)
	return out
}

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

// canonicalHostHeader matches typical browser/curl behavior: omit default port from the Host header.
// Some CDNs (e.g. behind certain load rules) answer differently for "example.com:80" vs "example.com".
func canonicalHostHeader(scheme, host, port string) string {
	if (scheme == "http" && port == "80") || (scheme == "https" && port == "443") {
		if ip := net.ParseIP(host); ip != nil && ip.To4() == nil {
			return "[" + host + "]"
		}
		return host
	}
	if port == "" {
		if ip := net.ParseIP(host); ip != nil && ip.To4() == nil {
			return "[" + host + "]"
		}
		return host
	}
	return net.JoinHostPort(host, port)
}

// prepareFasthttpRequest uses origin-form request targets (GET /path HTTP/1.1) with a
// canonical Host header. Absolute-form lines (GET http://host:80/ ...) confuse some CDNs
// even when Host omits the default port.
func prepareFasthttpRequest(req *fasthttp.Request, u *url.URL, method string) {
	scheme := u.Scheme
	host := u.Hostname()
	port := u.Port()
	if port == "" {
		if scheme == "https" {
			port = "443"
		} else {
			port = "80"
		}
	}
	reqPath := u.EscapedPath()
	if reqPath == "" {
		reqPath = "/"
	}
	if u.RawQuery != "" {
		reqPath += "?" + u.RawQuery
	}

	req.SetRequestURI(reqPath)
	req.Header.SetHost(canonicalHostHeader(scheme, host, port))
	req.UseHostHeader = true
	if scheme == "https" {
		req.URI().SetScheme("https")
	}
	req.Header.SetMethod(method)
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
		// Bare hostname or host:port (no scheme)
		if h, p, err := net.SplitHostPort(subdomain); err == nil {
			// Explicit port (e.g. docs.mercury.com:8443, go.mercury.com:80) — probe only that endpoint
			switch p {
			case "443":
				urls = []string{fmt.Sprintf("https://%s", net.JoinHostPort(h, p))}
			case "80":
				urls = []string{fmt.Sprintf("http://%s", net.JoinHostPort(h, p))}
			default:
				hp := net.JoinHostPort(h, p)
				urls = []string{
					fmt.Sprintf("https://%s", hp),
					fmt.Sprintf("http://%s", hp),
				}
			}
		} else if len(config.Ports) > 0 {
			// When 80 and/or 443 are in the list: only HTTPS if 443 is open, else only HTTP if 80 is open
			port443Open := isPortOpen(subdomain, "443")
			port80Open := isPortOpen(subdomain, "80")
			for _, port := range config.Ports {
				if port == "443" {
					if port443Open {
						urls = append(urls, fmt.Sprintf("https://%s:%s", subdomain, port))
					}
				} else if port == "80" {
					if !port443Open && port80Open {
						urls = append(urls, fmt.Sprintf("http://%s:%s", subdomain, port))
					}
				} else {
					// For non-standard ports, try both HTTP and HTTPS
					urls = append(urls, fmt.Sprintf("https://%s:%s", subdomain, port))
					urls = append(urls, fmt.Sprintf("http://%s:%s", subdomain, port))
				}
			}
		} else {
			// Default hostname only: only HTTPS if 443 is open, else only HTTP if 80 is open
			if isPortOpen(subdomain, "443") {
				urls = []string{fmt.Sprintf("https://%s", subdomain)}
			} else if isPortOpen(subdomain, "80") {
				urls = []string{fmt.Sprintf("http://%s", subdomain)}
			} else {
				urls = []string{} // both closed, nothing to probe
			}
		}
	}

	urls = prioritizeHTTPS(urls)

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

	// HTTPS-first: try candidates in list order; only move to the next URL if this one fails
	for _, targetURL := range urls {
		if result, ok := probeHTTP(client, targetURL, subdomain, config); ok {
			callback(result)
			break
		}
	}
}

// probeHTTP performs one GET against targetURL. Returns ok=true when any HTTP response was received.
func probeHTTP(client *fasthttp.Client, targetURL, subdomain string, config *Config) (Result, bool) {
	result := Result{
		URL:    targetURL,
		Input:  subdomain,
		Method: "GET",
		Failed: true,
	}

	parsedURL, err := url.Parse(targetURL)
	if err != nil {
		return result, false
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

	if !isPortOpen(host, port) {
		return result, false
	}

	req := fasthttp.AcquireRequest()
	resp := fasthttp.AcquireResponse()
	defer fasthttp.ReleaseRequest(req)
	defer fasthttp.ReleaseResponse(resp)

	prepareFasthttpRequest(req, parsedURL, "GET")
	req.Header.Set("User-Agent", "Mozilla/5.0")

	startTime := time.Now()

	requestTimeout := config.Timeout
	if requestTimeout > 5*time.Second {
		requestTimeout = 5 * time.Second
	}

	maxRetries := config.Retries
	if maxRetries < 0 {
		maxRetries = 0
	}

	var doErr error
	for attempt := 0; attempt <= maxRetries; attempt++ {
		doErr = client.DoTimeout(req, resp, requestTimeout)
		if doErr == nil {
			break
		}
		if attempt >= maxRetries {
			break
		}
		time.Sleep(time.Duration(attempt+1) * 50 * time.Millisecond)
	}

	if doErr != nil {
		return result, false
	}

	duration := time.Since(startTime)
	result.Time = duration.String()

	statusCode := resp.StatusCode()
	result.StatusCode = statusCode
	result.URL = targetURL
	result.Failed = false

	if parsedURL != nil {
		result.Scheme = parsedURL.Scheme
		result.Path = parsedURL.Path
		if result.Path == "" {
			result.Path = "/"
		}
		result.Port = port
	}

	domain := extractDomainFromURL(targetURL)

	result.ContentType = string(resp.Header.Peek("Content-Type"))
	result.Server = string(resp.Header.Peek("Server"))
	result.Location = string(resp.Header.Peek("Location"))

	headers := make(map[string]string)
	wappalyzerHeaders := make(map[string][]string)
	resp.Header.VisitAll(func(key, value []byte) {
		keyStr := string(key)
		valueStr := string(value)
		headers[keyStr] = valueStr
		wappalyzerHeaders[strings.ToLower(keyStr)] = []string{valueStr}
	})

	contentLength := resp.Header.ContentLength()
	if contentLength > 0 {
		result.ContentLength = int64(contentLength)
	} else {
		body := resp.Body()
		result.ContentLength = int64(len(body))
	}

	body := resp.Body()
	bodyStr := string(body)

	maxBodySize := 8192
	bodyForAnalysis := body
	if len(body) > maxBodySize {
		bodyForAnalysis = body[:maxBodySize]
	}
	bodyStrForAnalysis := string(bodyForAnalysis)

	if config.JSON || config.ShowHash || config.ShowTitle {
		if config.JSON || config.ShowHash {
			hashType := config.HashType
			if hashType == "" {
				hashType = "sha256"
			}

			if config.ShowHash && hashType == "md5" {
				result.Hash.BodyMD5 = calculateHash(body, "md5")

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

	if config.ShowFavicon {
		faviconHash := getFaviconHash(targetURL, client, config.Timeout)
		if faviconHash != "" {
			result.FaviconHash = faviconHash
		}
	}

	if config.JSON {
		result.Timestamp = time.Now().Format(time.RFC3339Nano)

		if domain != "" {
			result.IPs = resolveAllIPs(domain)
			if len(result.IPs) > 0 {
				result.Host = result.IPs[0]
				result.IP = result.IPs[0]

				cdnName, cdnType, isCDN := detectCDN(result.IPs[0], headers)
				if isCDN {
					result.CDNName = cdnName
					result.CDNType = cdnType
					result.CDN = true
				}
			}

			_, cname := resolveDNS(domain)
			result.CNAME = cname
		}

		result.Resolvers = getResolvers()

		words := strings.Fields(bodyStr)
		result.Words = len(words)
		lines := strings.Split(bodyStr, "\n")
		if len(lines) > 0 && lines[len(lines)-1] == "" {
			result.Lines = len(lines) - 1
		} else {
			result.Lines = len(lines)
		}

		result.Tech = detectTechnologies(targetURL, wappalyzerHeaders, bodyForAnalysis)

		pageType := getPageType(result.ContentType, bodyStrForAnalysis, result.StatusCode)
		if pageType == "" {
			pageType = "other"
		}
		pHash := calculatePHash(bodyStr, result.StatusCode)
		result.Knowledgebase = Knowledgebase{
			PageType: pageType,
			PHash:    pHash,
		}
	} else {
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

	return result, true
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
