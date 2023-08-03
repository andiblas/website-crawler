package crawler

import (
	"errors"
	"fmt"
	"net/url"
	"sync"

	"golang.org/x/net/context"

	"github.com/andiblas/website-crawler/pkg/fetcher"
	"github.com/andiblas/website-crawler/pkg/linkextractor"
)

type linkFoundCallback func(link url.URL)

type Crawler interface {
	Crawl(ctx context.Context, urlToCrawl url.URL, depth, maxConcurrency int, onNewLinksFound linkFoundCallback) ([]string, error)
}

type Concurrent struct {
	fetcher fetcher.Fetcher
}

func NewConcurrent(fetcher fetcher.Fetcher) *Concurrent {
	return &Concurrent{fetcher: fetcher}
}

// Crawl crawls a URL and returns a list of crawled links and any errors encountered.
// It uses a Concurrent crawler to crawl the URL and its linked pages concurrently.
// The recursionLimit argument how deep the crawler will continue retrieving links found in pages.
//
//	ctx := context.Background()
//	u, err := url.Parse("https://test.com")
//	linksFound, err := concurrent.Crawl(ctx, u, 2)
func (c *Concurrent) Crawl(ctx context.Context, urlToCrawl url.URL, recursionLimit int, onNewLinkFound linkFoundCallback) ([]string, error) {
	finishCh := make(chan bool)
	errorsCh := make(chan error)
	visitedLinksMap := sync.Map{}

	normalizedUrl := linkextractor.Normalize(urlToCrawl)
	go crawlerWorker(normalizedUrl, recursionLimit, onNewLinkFound, c.fetcher, &visitedLinksMap, errorsCh, finishCh)

	// listen for any errors that may occur while crawling in child crawlers
	var crawlingErrors []error
	for loop := true; loop; {
		select {
		case err := <-errorsCh:
			crawlingErrors = append(crawlingErrors, err)
		case <-ctx.Done():
			fmt.Println("crawling interrupted.")
			loop = false
		case <-finishCh:
			loop = false
		}
	}

	for _, err := range crawlingErrors {
		fmt.Println(err)
	}

	var crawledLinks []string
	visitedLinksMap.Range(func(key, value any) bool {
		crawledLinks = append(crawledLinks, key.(string))
		return true
	})

	if crawlingErrors != nil {
		return crawledLinks, errors.New("there were errors while crawling. please check logs")
	}

	return crawledLinks, nil
}

func crawlerWorker(urlToCrawl url.URL, recursionLimit int, onNewLinkFound linkFoundCallback, fetcher fetcher.Fetcher, visitedLinksMap *sync.Map, errorsCh chan error, finishCh chan bool) {
	if _, loaded := visitedLinksMap.LoadOrStore(urlToCrawl.String(), true); loaded {
		finishCh <- true
		return
	}

	if onNewLinkFound != nil {
		onNewLinkFound(urlToCrawl)
	}

	if recursionLimit <= 0 {
		finishCh <- true
		return
	}

	links, err := crawlWebpage(fetcher, urlToCrawl)
	if err != nil {
		visitedLinksMap.Delete(urlToCrawl.String())
		errorsCh <- fmt.Errorf("error while crawling [%s]: %w", urlToCrawl.String(), err)
		finishCh <- true
		return
	}

	childFinishChannel := make(chan bool)
	for _, link := range links {
		go crawlerWorker(link, recursionLimit-1, onNewLinkFound, fetcher, visitedLinksMap, errorsCh, childFinishChannel)
	}

	for i := 0; i < len(links); i++ {
		<-childFinishChannel
	}
	close(childFinishChannel)

	finishCh <- true
}

func crawlWebpage(httpFetcher fetcher.Fetcher, webpageURL url.URL) ([]url.URL, error) {
	webpageReader, err := httpFetcher.FetchWebpageContent(webpageURL)
	if err != nil {
		return nil, err
	}
	defer webpageReader.Close()

	links, err := linkextractor.Extract(webpageURL, webpageReader)
	if err != nil {
		return nil, err
	}

	return links, nil
}

type BreadthFirstCrawler struct {
	fetcher fetcher.Fetcher
}

func NewBreadthFirstCrawler(fetcher fetcher.Fetcher) *BreadthFirstCrawler {
	return &BreadthFirstCrawler{fetcher: fetcher}
}

func (a *BreadthFirstCrawler) Crawl(ctx context.Context, urlToCrawl url.URL, depth, maxConcurrency int, linkCallback linkFoundCallback) ([]string, error) {
	visitedLinks := sync.Map{}

	crawlInner(ctx, []url.URL{urlToCrawl}, a.fetcher, &visitedLinks, depth, maxConcurrency, linkCallback)

	var crawledLinks []string
	visitedLinks.Range(func(key, value any) bool {
		crawledLinks = append(crawledLinks, key.(string))
		return true
	})

	return crawledLinks, nil
}

func crawlInner(ctx context.Context, treeNodes []url.URL, fetcher fetcher.Fetcher, visitedLinks *sync.Map, depth int, maxConcurrency int, linkCallback linkFoundCallback) {
	var totalReferencedLinksAtDepth []url.URL

	batches := buildBatches(treeNodes, maxConcurrency)
	for _, batch := range batches {

		if errors.Is(ctx.Err(), context.Canceled) {
			return
		}

		wg := sync.WaitGroup{}
		for _, linkInBatch := range batch {
			if _, loaded := visitedLinks.LoadOrStore(linkInBatch.String(), true); loaded {
				continue
			}

			wg.Add(1)
			if linkCallback != nil {
				linkCallback(linkInBatch)
			}

			go func(link url.URL) {
				defer wg.Done()
				links, _ := crawlWebpage(fetcher, link)
				totalReferencedLinksAtDepth = append(totalReferencedLinksAtDepth, links...)
			}(linkInBatch)
		}
		wg.Wait()
	}
	if depth-1 > 0 {
		crawlInner(ctx, totalReferencedLinksAtDepth, fetcher, visitedLinks, depth-1, 0, linkCallback)
	}
}

func buildBatches(treeNodes []url.URL, batchSize int) [][]url.URL {
	var result [][]url.URL
	for i := 0; i < len(treeNodes); i += batchSize {
		j := i + batchSize
		if j > len(treeNodes) {
			j = len(treeNodes)
		}
		result = append(result, treeNodes[i:j])
	}
	return result
}
