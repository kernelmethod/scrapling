package main

import (
	"flag"
	"log"
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
	}

	err := RunWorkers(
		flag.Arg(0),
		threads,
		maxDepth,
		crawlExternalDomains,
	)
	if err != nil {
		log.Fatal(err)
	}
}
