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

func CrawlURL(
	base *url.URL,
	printedFilter *cuckoo.Filter,
	printedFilterLock *sync.Mutex,
) ([]string, error) {
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
	printedFilterLock.Lock()
	for _, u := range hrefs {
		ok := printedFilter.InsertUnique([]byte(u))
		if ok {
			fmt.Println(u)
		}
	}
	printedFilterLock.Unlock()

	return hrefs, nil
}

func Crawl(
	baseURL string,
	crawlExternalDomains bool,
	tasks chan Task,
	nTasks *sync.WaitGroup,
	printedFilter *cuckoo.Filter,
	printedFilterLock *sync.Mutex,
	processedFilter *cuckoo.Filter,
	processedFilterLock *sync.Mutex,
) {
	parsedBaseURL, _ := url.Parse(baseURL)

	for {
		task, ok := <-tasks

		if !ok {
			// Channel has closed, so we can cancel the worker
			break
		}

		err := HandleTask(
			&task,
			parsedBaseURL,
			crawlExternalDomains,
			tasks,
			nTasks,
			printedFilter,
			printedFilterLock,
			processedFilter,
			processedFilterLock,
		)

		if err != nil {
			fmt.Fprintf(os.Stderr, "Error processing %s: %s\n", task.URL, err)
		}
	}
}

func HandleTask(
	task *Task,
	base *url.URL,
	crawlExternalDomains bool,
	tasks chan Task,
	nTasks *sync.WaitGroup,
	printedFilter *cuckoo.Filter,
	printedFilterLock *sync.Mutex,
	processedFilter *cuckoo.Filter,
	processedFilterLock *sync.Mutex,
) error {
	// At the end of the function, we signal that we've completed a new task,
	// regardless of the outcome.
	defer nTasks.Done()

	// Skip this URL if we've exceeded the maximum recursion depth
	if task.depth < 0 {
		return nil
	}

	// Skip this URL if we've already processed it before
	processedFilterLock.Lock()
	ok := processedFilter.InsertUnique([]byte(task.URL))
	processedFilterLock.Unlock()
	if !ok {
		return nil
	}

	// Perform additional checks on the URL that we're about to query
	// - Ensure that the URL domain is the same as the original domain
	//   if crawlExternalDomains is false.
	// - Ensure that the URL scheme is either http or https
	taskURL, err := url.Parse(task.URL)
	ok = (err == nil)
	ok = ok && (crawlExternalDomains || taskURL.Host == base.Host)
	ok = ok && (taskURL.Scheme == "https" || taskURL.Scheme == "http")
	if !ok {
		return err
	}

	// Now that all of the checks have passed, we can crawl the URL
	hrefs, err := CrawlURL(taskURL, printedFilter, printedFilterLock)
	if err != nil {
		return err
	}

	// Add all of the extracted links back onto the task queue
	nTasks.Add(len(hrefs))
	go func() {
		for _, u := range hrefs {
			tasks <- Task{u, task.depth - 1}
		}
	}()

	return nil
}

func RunWorkers(
	baseURL string,
	threads int,
	maxDepth int,
	crawlExternalDomains bool,
) {

	var nTasks sync.WaitGroup

	tasks := make(chan Task)
	nTasks.Add(1)
	// go func() { tasks <- Task{baseURL, maxDepth} }()
	go func() { tasks <- Task{baseURL, maxDepth} }()

	var printedFilterLock, processedFilterLock sync.Mutex
	printedFilter := cuckoo.NewFilter(1_000_000)
	processedFilter := cuckoo.NewFilter(1_000_000)
	for i := 0; i < threads; i++ {
		go Crawl(
			baseURL,
			crawlExternalDomains,
			tasks,
			&nTasks,
			printedFilter,
			&printedFilterLock,
			processedFilter,
			&processedFilterLock,
		)
	}

	// In the primary goroutine, we wait until there are no more
	// tasks to be completed. Then we signal that the workers can
	// stop running.
	nTasks.Wait()
	close(tasks)
}
