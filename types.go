package main

import "time"

// Config holds all configuration options for the tool
type Config struct {
	ShowStatusCode    bool
	ShowContentType   bool
	ShowHash          bool
	HashType          string // md5, sha256, etc.
	ShowTitle         bool
	ShowServer        bool
	ShowIP            bool
	ShowCNAME         bool
	ShowContentLength bool
	ShowFavicon       bool
	JSON              bool
	Update            bool
	Threads           int
	Timeout           time.Duration
	RateLimit         int // requests per second
	Retries           int
	Ports             []string // custom ports to scan
	InputFile         string
	URL               string
}

// Knowledgebase holds knowledgebase information
type Knowledgebase struct {
	PageType string `json:"PageType"`
	PHash    int    `json:"pHash"`
}

// Hash holds hash information (body and header)
type Hash struct {
	BodyMD5   string `json:"body_md5,omitempty"`
	HeaderMD5 string `json:"header_md5,omitempty"`
}

// Result holds the result of checking a subdomain/URL
type Result struct {
	// Original fields
	URL           string
	StatusCode    int
	ContentType   string
	Title         string
	Server        string
	IP            string
	CNAME         string
	ContentLength int64
	Error         error

	// JSON output fields
	Input        string        `json:"input,omitempty"`
	Timestamp    string        `json:"timestamp,omitempty"`
	CDNName      string        `json:"cdn_name,omitempty"`
	CDNType      string        `json:"cdn_type,omitempty"`
	Port         string        `json:"port,omitempty"`
	Scheme       string        `json:"scheme,omitempty"`
	Path         string        `json:"path,omitempty"`
	Host         string        `json:"host,omitempty"`
	Method       string        `json:"method,omitempty"`
	Time         string        `json:"time,omitempty"`
	Location     string        `json:"location,omitempty"`
	IPs          []string      `json:"a,omitempty"`
	Words        int           `json:"words,omitempty"`
	Lines        int           `json:"lines,omitempty"`
	Tech         []string      `json:"tech,omitempty"`
	Failed       bool          `json:"failed,omitempty"`
	CDN          bool          `json:"cdn,omitempty"`
	FaviconHash  string        `json:"favicon,omitempty"`
	Hash         Hash          `json:"hash,omitempty"`
	Knowledgebase Knowledgebase `json:"knowledgebase,omitempty"`
	Resolvers    []string      `json:"resolvers,omitempty"`
}
