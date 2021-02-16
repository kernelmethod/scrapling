// Code for scraping URLs from parsed HTML

package main

import (
	"golang.org/x/net/html"
	"net/url"
)

// Convert an absolute or relative path into a full URL
func convertPath(href string, base *url.URL) (string, error) {
	parsedHref, err := url.Parse(href)
	if err != nil {
		return "", err
	}

	// If the scheme is empty, then the href must be an absolute
	// or relative path
	if parsedHref.Scheme == "" {
		return base.ResolveReference(parsedHref).String(), nil
	} else {
		return parsedHref.String(), nil
	}
}

// Extract all hrefs from <a>...</a> tags within an HTML response.
func extractHrefs(node *html.Node, base *url.URL) []string {
	var hrefs = []string{}

	// Check whether the node corresponds to an <a> tag. If it is, we
	// add its href attribute to the list
	if node.Type == html.ElementNode && node.Data == "a" {
		for _, attr := range node.Attr {
			if attr.Key == "href" {
				href, err := convertPath(attr.Val, base)
				if err != nil {
					// For some reason we experienced an error trying to parse this href,
					// so we skip over it
					break
				}
				hrefs = append(hrefs, href)
			}
		}
	}

	// Recursively check siblings and child nodes to see if we can
	// extract URLs from any of them
	for child := node.FirstChild; child != nil; child = child.NextSibling {
		hrefs = append(hrefs, extractHrefs(child, base)...)
	}

	return hrefs
}
