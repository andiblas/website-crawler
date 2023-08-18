package crawler

import (
	"errors"
	"fmt"
	"io"
	"net/url"
	"sync"

	"golang.org/x/net/context"

	"github.com/andiblas/website-crawler/pkg/fetcher"
	"github.com/andiblas/website-crawler/pkg/linkextractor"
)

// InvalidDepth indicates that the provided depth for the crawl operation is invalid.
// The depth value must be greater than 0.
var InvalidDepth = errors.New("invalid depth. must be greater than 0")

// InvalidMaxConcurrency indicates that the provided maximum concurrency value for
// the crawl operation is invalid. The maxConcurrency value must be greater than 0
// to allow concurrent crawling of multiple pages.
var InvalidMaxConcurrency = errors.New("invalid maximum concurrency. must be greater than 0")

type linkFoundCallback func(link url.URL)
type crawlingErrorCallback func(link url.URL, err error)

type Crawler interface {
	Crawl(ctx context.Context, urlToCrawl url.URL, depth, maxConcurrency int, linkFound linkFoundCallback, errorCallback crawlingErrorCallback) ([]string, error)
}

type BreadthFirstCrawler struct {
	fetcher fetcher.Fetcher
}

func NewBreadthFirstCrawler(fetcher fetcher.Fetcher) *BreadthFirstCrawler {
	return &BreadthFirstCrawler{fetcher: fetcher}
}

// Crawl performs a breadth-first web crawling starting from the specified URL.
// It explores the web pages up to the specified depth and concurrently crawls
// multiple pages based on the given maxConcurrency. The linkCallback function
// is executed each time a new link is discovered.
//
// The function takes the following parameters:
//   - ctx: Context that can be used to cancel the crawl operation.
//   - urlToCrawl: The initial URL from which the crawl will start.
//   - depth: The maximum depth of web page exploration during the crawl.
//   - maxConcurrency: The maximum number of pages to crawl concurrently.
//   - linkCallback: A function that will be called for each link found during the crawl.
//     It is called asynchronously for each link.
//
// The function returns an array of crawled URLs and an error. The crawled URLs
// are URLs that have been successfully visited during the crawl process. If an
// error occurs during the crawl, it is returned as an error value.
//
// If the provided depth is zero or negative, the function returns an error of type crawler.InvalidDepth.
// If the provided maxConcurrency is zero or negative, the function returns an error of type crawler.InvalidMaxConcurrency.
//
// The function uses breadth-first crawling to explore web pages and ensures that
// no duplicate URLs are visited. It also gracefully cancels the crawl if the provided
// context is canceled, allowing for clean shutdown of the crawling process.
//
// Please note that the linkFoundCallback and crawlingErrorCallback functions are executed
// in separate goroutines to do not hinder the main crawling process.
//
// Example usage:
//
//	crawler := crawler.NewBreadthFirstCrawler(fetcher)
//	urlToCrawl, _ := url.Parse("https://example.com")
//	depth := 3
//	maxConcurrency := 10
//	crawledLinks, err := crawler.Crawl(context.Background(), *urlToCrawl, depth, maxConcurrency, myLinkCallback)
//	if err != nil {
//	    fmt.Println("Error occurred during the crawl:", err)
//	} else {
//	    fmt.Println("Crawled links:", crawledLinks)
//	}
func (a *BreadthFirstCrawler) Crawl(ctx context.Context, urlToCrawl url.URL, depth, maxConcurrency int, linkFound linkFoundCallback, errorCallback crawlingErrorCallback) ([]string, error) {
	if depth <= 0 {
		return nil, InvalidDepth
	}
	if maxConcurrency <= 0 {
		return nil, InvalidMaxConcurrency
	}

	visitedLinks := sync.Map{}
	linksAtDepth := []url.URL{linkextractor.Normalize(urlToCrawl)}

	for currentDepth := 0; currentDepth < depth; currentDepth++ {
		batches := buildBatches(linksAtDepth, maxConcurrency)
		linksAtDepth = nil
		for _, batch := range batches {
			// graceful cancel before starting a new batch
			if errors.Is(ctx.Err(), context.Canceled) {
				break
			}

			linksAtDepth = append(linksAtDepth, crawlBatchConcurrently(batch, &visitedLinks, a.fetcher, linkFound, errorCallback)...)
		}
	}

	var crawledLinks []string
	visitedLinks.Range(func(key, value any) bool {
		crawledLinks = append(crawledLinks, key.(string))
		return true
	})

	return crawledLinks, nil
}

func crawlBatchConcurrently(batch []url.URL, visitedLinks *sync.Map, fetcher fetcher.Fetcher, linkFound linkFoundCallback, errorCallback crawlingErrorCallback) []url.URL {
	var result []url.URL
	wg := sync.WaitGroup{}
	for _, linkInBatch := range batch {
		if _, linkExists := visitedLinks.LoadOrStore(linkInBatch.String(), true); linkExists {
			continue
		}

		wg.Add(1)
		safeLinkFoundCallback(linkFound, linkInBatch)

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
