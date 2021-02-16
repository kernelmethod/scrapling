// scrape.go tests

package main

import (
	"golang.org/x/net/html"
	"net/url"
	"reflect"
	"sort"
	"strings"
	"testing"
)

func TestExtractHrefs(t *testing.T) {
	test := func(content string, baseURL *url.URL, expectedResults []string) {
		doc, err := html.Parse(strings.NewReader(content))
		if err != nil {
			t.Errorf("%s", err)
		}

		results := extractHrefs(doc, baseURL)
		sort.Strings(results)
		sort.Strings(expectedResults)

		if !reflect.DeepEqual(results, expectedResults) {
			t.Errorf("%s != %s", results, expectedResults)
		}
	}

	content := `
	<a href="https://www.example.org/foo">hello, world!</a>
	<a href="/bar">goodbye!</a>
	<img src="https://www.example.org/my/img">
	`
	baseURL, _ := url.Parse("https://www.example.org")
	hrefs := []string{"https://www.example.org/foo", "https://www.example.org/bar"}
	test(content, baseURL, hrefs)

	content = `
	<!doctype html>
	<html>
		<body>
			<p>Welcome to <a href="/">my home page!</a></p>
			<p>Here's a list of some of my favorite links:</p>
			<ul>
				<li><a href="https://www.google.com">My favorite search engine</a></li>
				<li><a href="/about">About me!</a></li>
				<li><a href="file:///etc/passwd">My favorite file :)</a></li>
			</ul>
			<p>And <a href="mailto:foobar@example.org">here's my email address!</a></p>
		</body>
	</html>
	`
	hrefs = []string{
		"https://www.google.com",
		"https://www.example.org/",
		"https://www.example.org/about",
		"file:///etc/passwd",
		"mailto:foobar@example.org",
	}
	test(content, baseURL, hrefs)
}
