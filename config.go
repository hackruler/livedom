package main

import (
	"flag"
	"strings"
	"time"
)

// parseFlags parses command line flags and returns a Config struct
func parseFlags() *Config {
	config := &Config{}

	flag.BoolVar(&config.ShowStatusCode, "sc", false, "Show status code")
	flag.BoolVar(&config.ShowContentType, "ct", false, "Show content type")
	// Hash flag is now a string to specify hash type
	flag.BoolVar(&config.ShowTitle, "title", false, "Show page title")
	flag.BoolVar(&config.ShowServer, "server", false, "Show server name")
	flag.BoolVar(&config.ShowIP, "ip", false, "Show IP address")
	flag.BoolVar(&config.ShowCNAME, "cname", false, "Show CNAME")
	flag.BoolVar(&config.ShowContentLength, "cl", false, "Show content length")
	flag.BoolVar(&config.ShowFavicon, "favicon", false, "Show favicon hash")
	flag.BoolVar(&config.JSON, "json", false, "Output results in JSON format")
	flag.BoolVar(&config.Update, "up", false, "Update livedom to the latest version")
	flag.IntVar(&config.Threads, "threads", 50, "Number of concurrent threads")
	var timeoutSeconds int
	flag.IntVar(&timeoutSeconds, "t", 5, "Request timeout in seconds")
	flag.IntVar(&config.RateLimit, "rl", 0, "Rate limit (requests per second, 0 = unlimited)")
	flag.IntVar(&config.Retries, "retries", 0, "Number of retries for failed requests")
	flag.StringVar(&config.HashType, "hash", "", "Show response body hash (md5, sha256, sha512, etc.)")
	flag.StringVar(&config.InputFile, "l", "", "Input file with subdomains (default: stdin)")
	flag.StringVar(&config.URL, "u", "", "Single URL/domain to check")
	
	// Ports flag - can accept comma-separated values or be called multiple times
	var portsFlag string
	flag.StringVar(&portsFlag, "p", "", "Ports to probe (comma-separated, e.g., 80,443,8080)")
	
	flag.Parse()
	
	// Parse ports if provided
	if portsFlag != "" {
		ports := strings.Split(portsFlag, ",")
		for _, port := range ports {
			port = strings.TrimSpace(port)
			if port != "" {
				config.Ports = append(config.Ports, port)
			}
		}
	}
	
	// If hash flag is set, enable ShowHash
	if config.HashType != "" {
		config.ShowHash = true
	}

	// Convert timeout seconds to duration
	config.Timeout = time.Duration(timeoutSeconds) * time.Second

	return config
}
