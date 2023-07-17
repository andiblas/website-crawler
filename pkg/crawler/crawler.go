package crawler

import (
	"errors"
	"fmt"
	"net/url"
	"sync"

	"github.com/andiblas/website-crawler/pkg/fetcher"
	"github.com/andiblas/website-crawler/pkg/linkextractor"
)

type Crawler interface {
	Crawl(url url.URL) ([]string, error)
}

type Concurrent struct {
	fetcher fetcher.Fetcher
}

func NewConcurrent(fetcher fetcher.Fetcher) *Concurrent {
	return &Concurrent{fetcher: fetcher}
}

func (c *Concurrent) Crawl(urlToCrawl url.URL) ([]string, error) {
	finishCh := make(chan bool)
	errorsCh := make(chan error)
	visitedLinksMap := sync.Map{}

	go crawlerWorker(linkextractor.Normalize(urlToCrawl), c.fetcher, &visitedLinksMap, errorsCh, finishCh)

	// listen for any errors that may occur while crawling in child crawlers
	var crawlingErrors []error
	for loop := true; loop; {
		select {
		case err := <-errorsCh:
			crawlingErrors = append(crawlingErrors, err)
		case <-finishCh:
			loop = false
			break
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
		return crawledLinks, errors.New("there were errors while crawling")
	}

	return crawledLinks, nil
}

func crawlerWorker(urlToCrawl url.URL, fetcher fetcher.Fetcher, visitedLinksMap *sync.Map, errorsCh chan error, finishCh chan bool) {
	if _, ok := visitedLinksMap.Load(urlToCrawl.String()); ok != false {
		finishCh <- true
		return
	}

	visitedLinksMap.Store(urlToCrawl.String(), true)
	links, err := crawlWebpage(fetcher, urlToCrawl)
	if err != nil {
		visitedLinksMap.Delete(urlToCrawl.String())
		errorsCh <- err
		finishCh <- true
		return
	}

	childFinishChannel := make(chan bool)
	for _, link := range links {
		go crawlerWorker(link, fetcher, visitedLinksMap, errorsCh, childFinishChannel)
	}

	for i := 0; i < len(links); i++ {
		<-childFinishChannel
	}

	finishCh <- true
}

func crawlWebpage(httpFetcher fetcher.Fetcher, webpageURL url.URL) ([]url.URL, error) {
	webpageContent, err := httpFetcher.FetchWebpageContent(webpageURL)
	if err != nil {
		return nil, err
	}

	links, err := linkextractor.Extract(webpageURL, webpageContent)
	if err != nil {
		return nil, err
	}

	return links, nil
}
