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

// Encapsulates a single task that must be completed by the worker
// goroutines.
type Task struct {
	URL   string
	depth int
}

// A worker that crawls URLs and extracts new links from them
type Worker struct {
	originalURL          *url.URL
	crawlExternalDomains bool
	tasks                chan Task
	nTasks               *sync.WaitGroup
	printedFilter        *cuckoo.Filter
	printedFilterLock    *sync.Mutex
	processedFilter      *cuckoo.Filter
	processedFilterLock  *sync.Mutex
}

func (w *Worker) CrawlURL(base *url.URL) ([]string, error) {
	resp, err := http.Get(base.String())
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

func (w *Worker) Crawl() {
	for {
		task, ok := <-w.tasks

		if !ok {
			// Channel has closed, so we can cancel the worker
			break
		}

		err := w.HandleTask(&task)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error processing %s: %s\n", task.URL, err)
		}
	}
}

func (w *Worker) HandleTask(task *Task) error {
	// At the end of the function, we signal that we've completed a new task,
	// regardless of the outcome.
	defer w.nTasks.Done()

	// Skip this URL if we've exceeded the maximum recursion depth
	if task.depth < 0 {
		return nil
	}

	// Skip this URL if we've already processed it before
	w.processedFilterLock.Lock()
	ok := w.processedFilter.InsertUnique([]byte(task.URL))
	w.processedFilterLock.Unlock()
	if !ok {
		return nil
	}

	// Perform additional checks on the URL that we're about to query
	// - Ensure that the URL domain is the same as the original domain
	//   if crawlExternalDomains is false.
	// - Ensure that the URL scheme is either http or https
	taskURL, err := url.Parse(task.URL)
	ok = (err == nil)
	ok = ok && (w.crawlExternalDomains || taskURL.Host == w.originalURL.Host)
	ok = ok && (taskURL.Scheme == "https" || taskURL.Scheme == "http")
	if !ok {
		return err
	}

	// Now that all of the checks have passed, we can crawl the URL
	hrefs, err := w.CrawlURL(taskURL)
	if err != nil {
		return err
	}

	// Add all of the extracted links back onto the task queue
	w.nTasks.Add(len(hrefs))
	go func() {
		for _, u := range hrefs {
			w.tasks <- Task{u, task.depth - 1}
		}
	}()

	return nil
}

func RunWorkers(
	baseURL string,
	threads int,
	maxDepth int,
	crawlExternalDomains bool,
) error {

	var nTasks sync.WaitGroup

	tasks := make(chan Task)
	nTasks.Add(1)
	// go func() { tasks <- Task{baseURL, maxDepth} }()
	go func() { tasks <- Task{baseURL, maxDepth} }()

	var printedFilterLock, processedFilterLock sync.Mutex
	printedFilter := cuckoo.NewFilter(1_000_000)
	processedFilter := cuckoo.NewFilter(1_000_000)
	parsedBaseURL, err := url.Parse(baseURL)
	if err != nil {
		return err
	}

	worker := Worker{
		parsedBaseURL,
		crawlExternalDomains,
		tasks,
		&nTasks,
		printedFilter,
		&printedFilterLock,
		processedFilter,
		&processedFilterLock,
	}
	for i := 0; i < threads; i++ {
		go worker.Crawl()
	}

	// In the primary goroutine, we wait until there are no more
	// tasks to be completed. Then we signal that the workers can
	// stop running.
	nTasks.Wait()
	close(tasks)

	return nil
}
