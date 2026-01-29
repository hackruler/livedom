package main

import (
	"io"
	"strings"

	"golang.org/x/net/html"
)

// extractTitle extracts the title from an HTML document
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
