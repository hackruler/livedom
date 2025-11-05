package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/fatih/color"
	"github.com/valyala/fasthttp"
)

type Config struct {
	ShowStatusCode bool
	Threads        int
	Timeout        time.Duration
	InputFile      string
}

type Result struct {
	URL        string
	StatusCode int
	Error      error
}

func main() {
	config := parseFlags()

	// Process subdomains as they come in (streaming)
	processSubdomainsStreaming(config)
}

func parseFlags() *Config {
	config := &Config{}

	flag.BoolVar(&config.ShowStatusCode, "sc", false, "Show status code")
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

	for _, url := range urls {
		req := fasthttp.AcquireRequest()
		resp := fasthttp.AcquireResponse()
		defer fasthttp.ReleaseRequest(req)
		defer fasthttp.ReleaseResponse(resp)

		req.SetRequestURI(url)
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
		result.URL = url
		return result
	}

	result.Error = fmt.Errorf("no response from HTTP or HTTPS")
	return result
}

func displaySingleResult(result Result, config *Config) {
	// Basic output - just the live subdomain
	if !config.ShowStatusCode {
		fmt.Println(color.New(color.FgWhite).Sprint(result.URL))
		return
	}

	// Extended output with status code
	statusColor := getStatusColor(result.StatusCode)
	fmt.Printf("%s %s\n",
		color.New(color.FgWhite).Sprint(result.URL),
		statusColor(fmt.Sprintf("[%d]", result.StatusCode)))
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
