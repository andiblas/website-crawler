package crawler

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/url"
	"sync"

	"github.com/andiblas/website-crawler/pkg/fetcher"
	"github.com/andiblas/website-crawler/pkg/linkextractor"
)

type linkFoundCallback func(link url.URL)
type crawlingErrorCallback func(link url.URL, err error)

type BreadthFirstCrawler struct {
	fetcher   fetcher.Fetcher
	linkFound linkFoundCallback
	onError   crawlingErrorCallback
}

// NewBreadthFirstCrawler creates a new breadth first crawler with the given fetcher and options.
//
// Parameters:
//   - fetcher: The fetcher implementation used to retrieve webpage content.
//   - opts: Optional variadic list of functional options to configure the crawler.
//
// Returns:
//   - A new instance of BreadthFirstCrawler initialized with the provided fetcher and options.
//
// Example:
//
//	fetcher := &MyFetcher{} // Replace with your fetcher implementation
//	crawler := NewBreadthFirstCrawler(fetcher, WithLinkFoundCallback(myLinkFoundCallback), WithOnErrorCallback(myErrorCallback))
func NewBreadthFirstCrawler(fetcher fetcher.Fetcher, opts ...Option) *BreadthFirstCrawler {
	bfc := &BreadthFirstCrawler{fetcher: fetcher}

	for _, opt := range opts {
		opt(bfc)
	}

	return bfc
}

// Crawl performs a breadth-first web crawling starting from the specified URL.
// It explores the web pages up to the specified depth and concurrently crawls
// multiple pages based on the given maxConcurrency. The linkCallback function
// is executed each time a new link is discovered.
//
// Parameters:
//   - ctx: The context used for cancellation and managing the crawl operation.
//   - urlToCrawl: The initial URL from which the crawl will start.
//   - depth: The maximum depth of web page exploration during the crawl.
//   - maxConcurrency: The maximum number of pages to crawl concurrently.
//
// Returns:
//   - An array of crawled URLs and an error. The crawled URLs are URLs that have
//     been found during the crawl process. The returned errors are for validation
//     purposes only. If you need to read an error while crawling a page, use the
//     WithOnErrorCallback option at the time of building this crawler.
//
// Errors:
//   - If the provided depth is zero or negative, the function returns an error of type InvalidDepth.
//   - If the provided maxConcurrency is zero or negative, the function returns an error of type InvalidMaxConcurrency.
//
// The function uses breadth-first crawling to explore web pages and ensures that
// no duplicate URLs are visited. It also gracefully cancels the crawl if the provided
// context is canceled, allowing for clean shutdown of the crawling process.
//
// The linkFoundCallback and crawlingErrorCallback functions are executed asynchronously
// in separate goroutines to avoid hindering the main crawling process.
//
// Example usage:
//
//	fetcher := &MyFetcher{} // Replace with your fetcher implementation
//	crawler := NewBreadthFirstCrawler(fetcher)
//	urlToCrawl, _ := url.Parse("https://example.com")
//	depth := 3
//	maxConcurrency := 10
//	crawledLinks, err := crawler.Crawl(context.Background(), *urlToCrawl, depth, maxConcurrency)
//	if err != nil {
//	    fmt.Println("Error occurred during the crawl:", err)
//	} else {
//	    fmt.Println("Crawled links:", crawledLinks)
//	}
func (bfc *BreadthFirstCrawler) Crawl(ctx context.Context, urlToCrawl url.URL, depth, maxConcurrency int) ([]string, error) {
	if depth <= 0 {
		return nil, InvalidDepth
	}
	if maxConcurrency <= 0 {
		return nil, InvalidMaxConcurrency
	}

	visitedLinks := make(map[string]bool) // map of links found while crawling + whether is visited or not
	linksAtDepth := []url.URL{linkextractor.Normalize(urlToCrawl)}

	for currentDepth := 0; currentDepth < depth; currentDepth++ {
		batches := buildBatches(linksAtDepth, maxConcurrency)
		linksAtDepth = nil
		for _, batch := range batches {
			// graceful cancel before starting a new batch
			if errors.Is(ctx.Err(), context.Canceled) {
				break
			}

			linksAtDepth = append(linksAtDepth, crawlBatchConcurrently(batch, visitedLinks, bfc.fetcher, bfc.onError)...)
		}
		for _, link := range linksAtDepth {
			if _, ok := visitedLinks[link.String()]; !ok {
				visitedLinks[link.String()] = false
				safeLinkFoundCallback(bfc.linkFound, link)
			}
		}
	}

	var i int
	crawledLinks := make([]string, len(visitedLinks))
	for link := range visitedLinks {
		crawledLinks[i] = link
		i++
	}

	return crawledLinks, nil
}

func crawlBatchConcurrently(batch []url.URL, visitedLinks map[string]bool, fetcher fetcher.Fetcher, errorCallback crawlingErrorCallback) []url.URL {
	var result []url.URL
	wg := sync.WaitGroup{}
	for _, linkInBatch := range batch {
		if visitedLinks[linkInBatch.String()] {
			continue
		}
		visitedLinks[linkInBatch.String()] = true

		wg.Add(1)

		go func(link url.URL) {
			defer wg.Done()
			links, err := crawlWebpage(fetcher, link)
			if err != nil {
				safeCrawlingErrorCallback(errorCallback, link, err)
				return
			}
			result = append(result, links...)
		}(linkInBatch)
	}
	wg.Wait()
	return result
}

func buildBatches(urlsToCrawl []url.URL, batchSize int) [][]url.URL {
	var result [][]url.URL
	for i := 0; i < len(urlsToCrawl); i += batchSize {
		j := i + batchSize
		if j > len(urlsToCrawl) {
			j = len(urlsToCrawl)
		}
		result = append(result, urlsToCrawl[i:j])
	}
	return result
}

func crawlWebpage(httpFetcher fetcher.Fetcher, webpageURL url.URL) ([]url.URL, error) {
	webpageReader, err := httpFetcher.FetchWebpageContent(webpageURL)
	if err != nil {
		return nil, err
	}
	defer func(webpageReader io.ReadCloser) {
		_ = webpageReader.Close()
	}(webpageReader)

	links, err := linkextractor.Extract(webpageURL, webpageReader)
	if err != nil {
		return nil, err
	}

	return links, nil
}

func safeLinkFoundCallback(linkFound linkFoundCallback, link url.URL) {
	if linkFound == nil {
		return
	}
	go func(l url.URL) {
		defer func() {
			if err := recover(); err != nil {
				fmt.Println("[RECOVERED] recovered from linkFoundCallback")
			}
		}()
		linkFound(l)
	}(link)
}

func safeCrawlingErrorCallback(errorCallback crawlingErrorCallback, link url.URL, err error) {
	if errorCallback == nil {
		return
	}
	go func(l url.URL, e error) {
		defer func() {
			if err := recover(); err != nil {
				fmt.Println("[RECOVERED] recovered from errorCallback")
			}
		}()
		errorCallback(l, e)
	}(link, err)
}
