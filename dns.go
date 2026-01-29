package main

import (
	"bufio"
	"net"
	"os"
	"strings"
)

// resolveDNS resolves the IP address and CNAME for a given domain
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
		cnameValue := strings.TrimSuffix(cnames, ".")
		// Only return CNAME if it's different from the domain (actual CNAME record exists)
		// If it's the same, it means there's no CNAME record
		if cnameValue != domain && cnameValue != domain+"." {
			cname = cnameValue
		}
	}

	return ip, cname
}

// resolveAllIPs resolves all IP addresses for a given domain
func resolveAllIPs(domain string) []string {
	var ips []string
	ipAddrs, err := net.LookupIP(domain)
	if err == nil {
		for _, ip := range ipAddrs {
			// Filter out IPv6 if needed, or include all
			ips = append(ips, ip.String())
		}
	}
	return ips
}

// getResolvers returns the DNS resolvers being used
func getResolvers() []string {
	var resolvers []string

	// Try to read from /etc/resolv.conf on Unix systems
	file, err := os.Open("/etc/resolv.conf")
	if err == nil {
		defer file.Close()
		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			// Skip comments and empty lines
			if line == "" || strings.HasPrefix(line, "#") {
				continue
			}
			// Parse "nameserver 1.1.1.1" format
			if strings.HasPrefix(line, "nameserver ") {
				parts := strings.Fields(line)
				if len(parts) >= 2 {
					ip := parts[1]
					// Validate IP address
					if net.ParseIP(ip) != nil {
						resolvers = append(resolvers, ip+":53")
					}
				}
			}
		}
	}

	// If no resolvers found, use defaults
	if len(resolvers) == 0 {
		resolvers = []string{
			"1.1.1.1:53", // Cloudflare
			"8.8.8.8:53", // Google
		}
	}

	return resolvers
}
