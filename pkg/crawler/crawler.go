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

type BreadthFirstCrawler struct {
	fetcher fetcher.Fetcher
}

func NewBreadthFirstCrawler(fetcher fetcher.Fetcher) *BreadthFirstCrawler {
	return &BreadthFirstCrawler{fetcher: fetcher}
}

func (a *BreadthFirstCrawler) Crawl(ctx context.Context, urlToCrawl url.URL, depth, maxConcurrency int, linkCallback linkFoundCallback) ([]string, error) {
	visitedLinks := sync.Map{}

	crawlInner(ctx, []url.URL{linkextractor.Normalize(urlToCrawl)}, a.fetcher, &visitedLinks, depth, maxConcurrency, linkCallback)

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
		// graceful cancel before starting a new batch
		if errors.Is(ctx.Err(), context.Canceled) {
			return
		}

		wg := sync.WaitGroup{}
		for _, linkInBatch := range batch {
			if _, loaded := visitedLinks.LoadOrStore(linkInBatch.String(), true); loaded {
				continue
			}

			wg.Add(1)
			safeCallback(linkCallback, linkInBatch)

			go func(link url.URL) {
				defer wg.Done()
				links, err := crawlWebpage(fetcher, link)
				if err != nil {
					fmt.Printf("[ERROR] error while crawling [%s] err: %v\n", link.String(), err)
				}
				totalReferencedLinksAtDepth = append(totalReferencedLinksAtDepth, links...)
			}(linkInBatch)
		}
		wg.Wait()
	}
	if depth-1 > 0 {
		crawlInner(ctx, totalReferencedLinksAtDepth, fetcher, visitedLinks, depth-1, maxConcurrency, linkCallback)
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

func safeCallback(linkFound linkFoundCallback, link url.URL) {
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
