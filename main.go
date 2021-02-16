package main

import (
	"flag"
	"fmt"
	"golang.org/x/net/html"
	"log"
	"net/http"
	"os"
)

func main() {
	var external_domains bool
	var depth int

	// Command-line arguments
	flag.IntVar(&depth, "d", 0, "Recursion depth to go to")
	flag.BoolVar(&external_domains, "external-domains", false, "Recursively search URLs from other domains that are encountered")
	flag.Parse()

	if flag.NArg() != 1 {
		log.Fatal("Missing argument: start_url\n")
		os.Exit(1)
	}

	resp, err := http.Get(flag.Arg(0))
	if err != nil {
		log.Fatal(err)
	}
	defer resp.Body.Close()

	doc, err := html.Parse(resp.Body)
	if err != nil {
		log.Fatal(err)
	}

	hrefs := extractHrefs(doc)
	for _, u := range hrefs {
		fmt.Println(u)
	}
}
