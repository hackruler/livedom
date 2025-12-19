package main

import (
	"bufio"
	"crypto/sha256"
	"encoding/hex"
	"flag"
	"fmt"
	"io"
	"net"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/fatih/color"
	"github.com/valyala/fasthttp"
	"golang.org/x/net/html"
)

type Config struct {
	ShowStatusCode    bool
	ShowContentType   bool
	ShowHash          bool
	ShowTitle         bool
	ShowServer        bool
	ShowIP            bool
	ShowCNAME         bool
	ShowContentLength bool
	Threads           int
	Timeout           time.Duration
	InputFile         string
}

type Result struct {
	URL           string
	StatusCode    int
	ContentType   string
	Hash          string
	Title         string
	Server        string
	IP            string
	CNAME         string
	ContentLength int64
	Error         error
}

func main() {
	// Force color output even when redirecting to file
	// This ensures ANSI color codes are written to files (like httpx)
	color.NoColor = false
	// Override the output to always enable colors
	color.Output = os.Stdout

	config := parseFlags()

	// Process subdomains as they come in (streaming)
	processSubdomainsStreaming(config)
}

func parseFlags() *Config {
	config := &Config{}

	flag.BoolVar(&config.ShowStatusCode, "sc", false, "Show status code")
	flag.BoolVar(&config.ShowContentType, "ct", false, "Show content type")
	flag.BoolVar(&config.ShowHash, "hash", false, "Show response body hash")
	flag.BoolVar(&config.ShowTitle, "title", false, "Show page title")
	flag.BoolVar(&config.ShowServer, "server", false, "Show server name")
	flag.BoolVar(&config.ShowIP, "ip", false, "Show IP address")
	flag.BoolVar(&config.ShowCNAME, "cname", false, "Show CNAME")
	flag.BoolVar(&config.ShowContentLength, "cl", false, "Show content length")
	flag.IntVar(&config.Threads, "t", 50, "Number of concurrent threads")
	flag.DurationVar(&config.Timeout, "timeout", 5*time.Second, "Request timeout")
	flag.StringVar(&config.InputFile, "f", "", "Input file with subdomains (default: stdin)")

	flag.Parse()

	return config
}

func readSubdomains(inputFile string) []string {
	var reader io.Reader

	if inputFile != "" {
		file, err := os.Open(inputFile)
		if err != nil {
			fmt.Printf("Error opening file: %v\n", err)
			os.Exit(1)
		}
		defer file.Close()
		reader = file
	} else {
		reader = os.Stdin
	}

	var subdomains []string
	scanner := bufio.NewScanner(reader)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line != "" {
			subdomains = append(subdomains, line)
		}
	}

	if err := scanner.Err(); err != nil {
		fmt.Printf("Error reading input: %v\n", err)
		os.Exit(1)
	}

	return subdomains
}

func processSubdomainsStreaming(config *Config) {
	var reader io.Reader

	if config.InputFile != "" {
		file, err := os.Open(config.InputFile)
		if err != nil {
			fmt.Printf("Error opening file: %v\n", err)
			os.Exit(1)
		}
		defer file.Close()
		reader = file
	} else {
		reader = os.Stdin
	}

	scanner := bufio.NewScanner(reader)

	// Create worker pool
	semaphore := make(chan struct{}, config.Threads)
	var wg sync.WaitGroup

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line != "" {
			wg.Add(1)
			go func(subdomain string) {
				defer wg.Done()
				semaphore <- struct{}{}        // Acquire
				defer func() { <-semaphore }() // Release

				result := checkSubdomain(subdomain, config)
				if result.Error == nil {
					displaySingleResult(result, config)
				}
			}(line)
		}
	}

	if err := scanner.Err(); err != nil {
		fmt.Printf("Error reading input: %v\n", err)
		os.Exit(1)
	}

	// Wait for all workers to complete
	wg.Wait()
}

func processSubdomains(subdomains []string, config *Config) {
	resultChan := make(chan Result, len(subdomains))

	// Create worker pool
	semaphore := make(chan struct{}, config.Threads)
	var wg sync.WaitGroup

	for _, subdomain := range subdomains {
		wg.Add(1)
		go func(sub string) {
			defer wg.Done()
			semaphore <- struct{}{}        // Acquire
			defer func() { <-semaphore }() // Release

			result := checkSubdomain(sub, config)
			resultChan <- result
		}(subdomain)
	}

	// Close channel when all workers are done
	go func() {
		wg.Wait()
		close(resultChan)
	}()

	// Display results as they come in (streaming output like httpx)
	for result := range resultChan {
		if result.Error == nil {
			displaySingleResult(result, config)
		}
	}
}

func checkSubdomain(subdomain string, config *Config) Result {
	result := Result{URL: subdomain}

	// Check if input is already a full URL
	var urls []string
	if strings.HasPrefix(subdomain, "http://") || strings.HasPrefix(subdomain, "https://") {
		// It's already a full URL, use it directly
		urls = []string{subdomain}
	} else {
		// It's just a domain, try HTTPS first, then HTTP
		urls = []string{
			fmt.Sprintf("https://%s", subdomain),
			fmt.Sprintf("http://%s", subdomain),
		}
	}

	// Create fasthttp client with optimized settings
	client := &fasthttp.Client{
		MaxConnsPerHost:               200,
		MaxIdleConnDuration:           30 * time.Second,
		ReadTimeout:                   config.Timeout,
		WriteTimeout:                  config.Timeout,
		MaxIdemponentCallAttempts:     1,
		DisableHeaderNamesNormalizing: true,
		DisablePathNormalizing:        true,
	}

	for _, targetURL := range urls {
		req := fasthttp.AcquireRequest()
		resp := fasthttp.AcquireResponse()
		defer fasthttp.ReleaseRequest(req)
		defer fasthttp.ReleaseResponse(resp)

		req.SetRequestURI(targetURL)
		req.Header.SetMethod("GET")
		req.Header.Set("User-Agent", "Mozilla/5.0")

		err := client.Do(req, resp)
		if err != nil {
			continue // Try next URL
		}

		statusCode := resp.StatusCode()

		// Accept any response (including 4xx, 5xx) as "live"
		// This matches httpx behavior
		result.StatusCode = statusCode
		result.URL = targetURL

		// Extract domain from URL for DNS resolution
		parsedURL, _ := url.Parse(targetURL)
		domain := parsedURL.Hostname()

		// Get headers
		result.ContentType = string(resp.Header.Peek("Content-Type"))
		result.Server = string(resp.Header.Peek("Server"))
		
		// Get content length from header, or use body length as fallback
		contentLength := resp.Header.ContentLength()
		if contentLength > 0 {
			result.ContentLength = int64(contentLength)
		} else {
			// If Content-Length header is not present, use actual body size
			body := resp.Body()
			result.ContentLength = int64(len(body))
		}

		// Read response body if needed for hash or title
		// Limit to 8KB for performance
		body := resp.Body()
		maxBodySize := 8192
		if len(body) > maxBodySize {
			body = body[:maxBodySize]
		}

		if config.ShowHash || config.ShowTitle {
			if config.ShowHash {
				hash := sha256.Sum256(body)
				result.Hash = hex.EncodeToString(hash[:])
			}
			if config.ShowTitle {
				title, _ := extractTitle(strings.NewReader(string(body)))
				result.Title = title
			}
		}

		// Resolve IP and CNAME if needed
		if config.ShowIP || config.ShowCNAME {
			ip, cname := resolveDNS(domain)
			if config.ShowIP {
				result.IP = ip
			}
			if config.ShowCNAME {
				result.CNAME = cname
			}
		}

		return result
	}

	result.Error = fmt.Errorf("no response from HTTP or HTTPS")
	return result
}

func extractTitle(body io.Reader) (string, error) {
	doc, err := html.Parse(body)
	if err != nil {
		return "", err
	}

	var title string
	var findTitle func(*html.Node)
	findTitle = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "title" {
			if n.FirstChild != nil {
				title = n.FirstChild.Data
				return
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			if title != "" {
				return
			}
			findTitle(c)
		}
	}

	findTitle(doc)
	return strings.TrimSpace(title), nil
}

func resolveDNS(domain string) (string, string) {
	var ip string
	var cname string

	// Resolve IP
	ips, err := net.LookupIP(domain)
	if err == nil && len(ips) > 0 {
		ip = ips[0].String()
	}

	// Resolve CNAME
	cnames, err := net.LookupCNAME(domain)
	if err == nil && cnames != "" {
		cname = strings.TrimSuffix(cnames, ".")
	}

	return ip, cname
}

// Helper function to create color with colors always enabled
func newColor(attr color.Attribute) *color.Color {
	c := color.New(attr)
	c.EnableColor()
	return c
}

func displaySingleResult(result Result, config *Config) {
	var output []string

	// Always show URL
	output = append(output, newColor(color.FgWhite).Sprint(result.URL))

	// Status code
	if config.ShowStatusCode {
		statusColor := getStatusColor(result.StatusCode)
		output = append(output, statusColor(fmt.Sprintf("[%d]", result.StatusCode)))
	}

	// Content type
	if config.ShowContentType {
		if result.ContentType != "" {
			contentType := strings.Split(result.ContentType, ";")[0]
			output = append(output, color.New(color.FgYellow).Sprint(fmt.Sprintf("[%s]", contentType)))
		} else {
			output = append(output, color.New(color.FgYellow).Sprint("[]"))
		}
	}

	// Content length
	if config.ShowContentLength {
		if result.ContentLength > 0 {
			output = append(output, color.New(color.FgCyan).Sprint(fmt.Sprintf("[%d]", result.ContentLength)))
		} else {
			output = append(output, color.New(color.FgCyan).Sprint("[]"))
		}
	}

	// Hash
	if config.ShowHash {
		if result.Hash != "" {
			output = append(output, color.New(color.FgMagenta).Sprint(fmt.Sprintf("[%s]", result.Hash)))
		} else {
			output = append(output, color.New(color.FgMagenta).Sprint("[]"))
		}
	}

	// Title
	if config.ShowTitle {
		if result.Title != "" {
			title := truncateString(result.Title, 50)
			output = append(output, color.New(color.FgBlue).Sprint(fmt.Sprintf("[%s]", title)))
		} else {
			output = append(output, color.New(color.FgBlue).Sprint("[]"))
		}
	}

	// Server
	if config.ShowServer {
		if result.Server != "" {
			output = append(output, color.New(color.FgGreen).Sprint(fmt.Sprintf("[%s]", result.Server)))
		} else {
			output = append(output, color.New(color.FgGreen).Sprint("[]"))
		}
	}

	// IP
	if config.ShowIP {
		if result.IP != "" {
			output = append(output, color.New(color.FgCyan).Sprint(fmt.Sprintf("[%s]", result.IP)))
		} else {
			output = append(output, color.New(color.FgCyan).Sprint("[]"))
		}
	}

	// CNAME
	if config.ShowCNAME {
		if result.CNAME != "" {
			output = append(output, color.New(color.FgYellow).Sprint(fmt.Sprintf("[%s]", result.CNAME)))
		} else {
			output = append(output, color.New(color.FgYellow).Sprint("[]"))
		}
	}

	// If no flags are set, just show URL
	// Use color.Output to ensure colors are written even when redirecting to file
	if len(output) == 1 {
		fmt.Fprintln(color.Output, output[0])
	} else {
		fmt.Fprintln(color.Output, strings.Join(output, " "))
	}
}

func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}

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
