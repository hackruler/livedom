package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	"time"
)

// processSubdomainsStreaming processes subdomains as they come in (streaming)
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

	// Rate limiting
	var rateLimiter *time.Ticker
	if config.RateLimit > 0 {
		rateLimiter = time.NewTicker(time.Second / time.Duration(config.RateLimit))
		defer rateLimiter.Stop()
	}

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line != "" {
			// Rate limiting: wait if rate limit is set
			if rateLimiter != nil {
				<-rateLimiter.C
			}

			wg.Add(1)
			go func(subdomain string) {
				defer wg.Done()
				semaphore <- struct{}{}        // Acquire
				defer func() { <-semaphore }() // Release

				checkSubdomain(subdomain, config, func(result Result) {
					if result.Error == nil {
						displaySingleResult(result, config)
					}
				})
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

// processSubdomains processes subdomains in batch mode (not currently used, but kept for reference)
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

			checkSubdomain(sub, config, func(result Result) {
				resultChan <- result
			})
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
