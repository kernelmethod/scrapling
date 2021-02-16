// scrape.go tests

package main

import (
	"net/url"
	"testing"
)

func TestConvertPath(t *testing.T) {
	test := func(href string, base *url.URL, expectedResult string) {
		result, err := convertPath(href, base)
		if err != nil {
			t.Errorf("convertPath error: %s", err)
		}
		if result != expectedResult {
			t.Errorf(
				"convertPath(\"%s\", url.Parse(\"%s\")) == \"%s\" != \"%s\"",
				href, base, result, expectedResult,
			)
		}
	}

	base, _ := url.Parse("https://www.example.org/foo")

	// Case 1: the href is a full URL. In this case convertPath should do
	// nothing to the href
	test("https://www.example.com/bar", base, "https://www.example.com/bar")

	// Case 2: the href is an absolute path
	test("/bar", base, "https://www.example.org/bar")

	// Case 3: TODO: the href is a relative path
	// test("./bar", base, "https://www.example.org/foo/bar")
}
