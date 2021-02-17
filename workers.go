package main

import (
	"fmt"
	"github.com/seiflotfy/cuckoofilter"
	"golang.org/x/net/html"
	"net/http"
	"net/url"
	"os"
	"sync"
)

// A worker that crawls URLs and extracts new links from them
type Worker struct {
	originalURL          *url.URL
	crawlExternalDomains bool
	maxRequestsLock      chan struct{}
	nTasks               *sync.WaitGroup
	printedFilter        *cuckoo.Filter
	printedFilterLock    *sync.Mutex
	processedFilter      *cuckoo.Filter
	processedFilterLock  *sync.Mutex
}

// Have the worker make a GET request to a URL. Wraps some additional actions, such
// as grabbing the maxRequestsLock to ensure that we don't run more concurrent HTTP
// requests than we need.
func (w *Worker) HttpGet(base *url.URL) (*http.Response, error) {
	// Grab the maxRequestsLock semaphore
	w.maxRequestsLock <- struct{}{}
	resp, err := http.Get(base.String())

	// Release the maxRequestsLock semaphore
	<-w.maxRequestsLock

	return resp, err
}

func (w *Worker) ScrapeLinks(base *url.URL) ([]string, error) {
	resp, err := w.HttpGet(base)
	if err != nil {
		return []string{}, err
	}
	defer resp.Body.Close()

	doc, err := html.Parse(resp.Body)
	if err != nil {
		return []string{}, err
	}

	base = resp.Request.URL
	hrefs := extractHrefs(doc, base)

	// Print any hrefs that haven't been printed before
	w.printedFilterLock.Lock()
	for _, u := range hrefs {
		ok := w.printedFilter.InsertUnique([]byte(u))
		if ok {
			fmt.Println(u)
		}
	}
	w.printedFilterLock.Unlock()

	return hrefs, nil
}

func (w *Worker) Crawl(url string, depth int) {
	// At the end of the function, we signal that we've completed a new task,
	// regardless of the outcome.
	defer w.nTasks.Done()

	err := w.HandleTask(url, depth)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error processing %s: %s\n", url, err)
	}
}

func (w *Worker) HandleTask(taskURL string, depth int) error {
	// Skip this URL if we've exceeded the maximum recursion depth
	if depth < 0 {
		return nil
	}

	// Skip this URL if we've already processed it before
	w.processedFilterLock.Lock()
	ok := w.processedFilter.InsertUnique([]byte(taskURL))
	w.processedFilterLock.Unlock()
	if !ok {
		return nil
	}

	// Perform additional checks on the URL that we're about to query
	// - Ensure that the URL domain is the same as the original domain
	//   if crawlExternalDomains is false.
	// - Ensure that the URL scheme is either http or https
	parsedTaskURL, err := url.Parse(taskURL)
	ok = (err == nil)
	ok = ok && (w.crawlExternalDomains || parsedTaskURL.Host == w.originalURL.Host)
	ok = ok && (parsedTaskURL.Scheme == "https" || parsedTaskURL.Scheme == "http")
	if !ok {
		return err
	}

	// Now that all of the checks have passed, we can crawl the URL
	hrefs, err := w.ScrapeLinks(parsedTaskURL)
	if err != nil {
		return err
	}

	// Start new tasks for all of the scraped links
	w.nTasks.Add(len(hrefs))
	for _, u := range hrefs {
		go w.Crawl(u, depth-1)
	}

	return nil
}

func RunWorkers(
	baseURL string,
	threads int,
	maxDepth int,
	crawlExternalDomains bool,
) error {

	var nTasks sync.WaitGroup
	var printedFilterLock, processedFilterLock sync.Mutex

	maxRequestsLock := make(chan struct{}, threads)

	nTasks.Add(1)
	printedFilter := cuckoo.NewFilter(1_000_000)
	processedFilter := cuckoo.NewFilter(1_000_000)
	parsedBaseURL, err := url.Parse(baseURL)
	if err != nil {
		return err
	}

	worker := Worker{
		parsedBaseURL,
		crawlExternalDomains,
		maxRequestsLock,
		&nTasks,
		printedFilter,
		&printedFilterLock,
		processedFilter,
		&processedFilterLock,
	}
	go worker.Crawl(baseURL, maxDepth)

	// In the primary goroutine, we wait until there are no more
	// tasks to be completed. Then we signal that the workers can
	// stop running.
	nTasks.Wait()

	return nil
}
