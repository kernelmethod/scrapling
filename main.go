package main

import (
	"flag"
	_ "golang.org/x/net/html"
	"log"
	_ "net/http"
	"os"
)

var crawlExternalDomains bool
var maxDepth, threads int

func main() {
	// Command-line arguments
	flag.IntVar(&maxDepth, "d", 0, "Recursion depth to go to")
	flag.IntVar(&threads, "t", 10, "Maximum number of concurrent HTTP requests to process")
	flag.BoolVar(&crawlExternalDomains, "external-domains", false, "Recursively search URLs from other domains that are encountered")
	flag.Parse()

	if flag.NArg() != 1 {
		log.Fatal("Missing argument: start_url\n")
		os.Exit(1)
	}

	RunWorkers(
		flag.Arg(0),
		threads,
		maxDepth,
		crawlExternalDomains,
	)

	/*
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
	*/
}
