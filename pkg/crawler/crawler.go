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

type Crawler interface {
	Crawl(ctx context.Context, urlToCrawl url.URL, depth int) ([]string, error)
}

type Concurrent struct {
	fetcher fetcher.Fetcher
}

func NewConcurrent(fetcher fetcher.Fetcher) *Concurrent {
	return &Concurrent{fetcher: fetcher}
}

// Crawl crawls a URL and returns a list of crawled links and any errors encountered.
// It uses a Concurrent crawler to crawl the URL and its linked pages concurrently.
//
//	ctx := context.Background()
//	u, err := url.Parse("https://test.com")
//	linksFound, err := concurrent.Crawl(ctx, u)
func (c *Concurrent) Crawl(ctx context.Context, urlToCrawl url.URL, depth int) ([]string, error) {
	finishCh := make(chan bool)
	errorsCh := make(chan error)
	visitedLinksMap := sync.Map{}

	normalizedUrl := linkextractor.Normalize(urlToCrawl)
	relativeDepth := linkextractor.LinkDepth(normalizedUrl) + depth
	go crawlerWorker(normalizedUrl, relativeDepth, c.fetcher, &visitedLinksMap, errorsCh, finishCh)

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
		fmt.Printf("error occured while crawling: %+v\n", err)
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

func crawlerWorker(urlToCrawl url.URL, depth int, fetcher fetcher.Fetcher, visitedLinksMap *sync.Map, errorsCh chan error, finishCh chan bool) {
	if depth < linkextractor.LinkDepth(urlToCrawl) {
		finishCh <- true
		return
	}

	if _, loaded := visitedLinksMap.LoadOrStore(urlToCrawl.String(), true); loaded {
		finishCh <- true
		return
	}

	links, err := crawlWebpage(fetcher, urlToCrawl)
	if err != nil {
		visitedLinksMap.Delete(urlToCrawl.String())
		errorsCh <- err
		finishCh <- true
		return
	}

	childFinishChannel := make(chan bool)
	for _, link := range links {
		go crawlerWorker(link, depth, fetcher, visitedLinksMap, errorsCh, childFinishChannel)
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
