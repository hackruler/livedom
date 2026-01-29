package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/fatih/color"
)

// displaySingleResult displays a single result with formatted output
func displaySingleResult(result Result, config *Config) {
	// If JSON mode, output JSON
	if config.JSON {
		displayJSONResult(result)
		return
	}

	var output []string

	// Always show URL (no color)
	output = append(output, result.URL)

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
		if result.Hash.BodyMD5 != "" {
			output = append(output, color.New(color.FgMagenta).Sprint(fmt.Sprintf("[%s]", result.Hash.BodyMD5)))
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

	// Favicon
	if config.ShowFavicon {
		if result.FaviconHash != "" {
			output = append(output, color.New(color.FgCyan).Sprint(fmt.Sprintf("[%s]", result.FaviconHash)))
		} else {
			output = append(output, color.New(color.FgCyan).Sprint("[]"))
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

// JSONOutput represents the JSON output structure matching httpx format
type JSONOutput struct {
	Timestamp    string        `json:"timestamp,omitempty"`
	CDNName      string        `json:"cdn_name,omitempty"`
	CDNType      string        `json:"cdn_type,omitempty"`
	Port         string        `json:"port,omitempty"`
	URL          string        `json:"url,omitempty"`
	Input        string        `json:"input,omitempty"`
	Location     string        `json:"location,omitempty"`
	Title        string        `json:"title,omitempty"`
	Scheme       string        `json:"scheme,omitempty"`
	Webserver    string        `json:"webserver,omitempty"`
	ContentType  string        `json:"content_type,omitempty"`
	Method       string        `json:"method,omitempty"`
	Host         string        `json:"host,omitempty"`
	Path         string        `json:"path,omitempty"`
	Time         string        `json:"time,omitempty"`
	IPs          []string      `json:"a,omitempty"`
	Tech         []string      `json:"tech,omitempty"`
	Words        int           `json:"words,omitempty"`
	Lines        int           `json:"lines,omitempty"`
	StatusCode   int           `json:"status_code,omitempty"`
	ContentLength int64        `json:"content_length,omitempty"`
	Failed       bool          `json:"failed,omitempty"`
	CDN          bool          `json:"cdn,omitempty"`
	FaviconHash  string        `json:"favicon,omitempty"`
	Hash         Hash          `json:"hash,omitempty"`
	Knowledgebase Knowledgebase `json:"knowledgebase,omitempty"`
	Resolvers    []string      `json:"resolvers,omitempty"`
}

// displayJSONResult outputs the result in JSON format
func displayJSONResult(result Result) {
	// Build JSON output struct matching httpx format
	jsonOutput := JSONOutput{
		Timestamp:     result.Timestamp,
		CDNName:       result.CDNName,
		CDNType:       result.CDNType,
		Port:          result.Port,
		URL:           result.URL,
		Input:         result.Input,
		Location:      result.Location,
		Title:         result.Title,
		Scheme:        result.Scheme,
		Webserver:     result.Server,
		ContentType:   result.ContentType,
		Method:        result.Method,
		Host:          result.Host,
		Path:          result.Path,
		Time:          result.Time,
		IPs:           result.IPs,
		Tech:          result.Tech,
		Words:         result.Words,
		Lines:         result.Lines,
		StatusCode:    result.StatusCode,
		ContentLength: result.ContentLength,
		Failed:        result.Failed,
		CDN:           result.CDN,
		FaviconHash:   result.FaviconHash,
		Hash:          result.Hash,
		Knowledgebase: result.Knowledgebase,
		Resolvers:     result.Resolvers,
	}

	// Ensure PageType is set
	if jsonOutput.Knowledgebase.PageType == "" {
		jsonOutput.Knowledgebase.PageType = "other"
	}

	// Marshal to JSON
	jsonData, err := json.Marshal(jsonOutput)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error marshaling JSON: %v\n", err)
		return
	}

	// Output JSON
	fmt.Println(string(jsonData))
}
