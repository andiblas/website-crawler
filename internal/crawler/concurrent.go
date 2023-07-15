package crawler

import (
	"fmt"
	"net/url"
	"sync"
	"time"

	"github.com/andiblas/website-crawler/internal/fetcher"
	"github.com/andiblas/website-crawler/internal/linkextractor"
)

const crawlerWorkerAmount = 5

type Concurrent struct {
	fetcher fetcher.Fetcher
}

func NewConcurrent(fetcher fetcher.Fetcher) *Concurrent {
	return &Concurrent{fetcher: fetcher}
}

func (c *Concurrent) Crawl(urlToCrawl url.URL) ([]string, error) {
	pendingURLch := make(chan string) // Channel to hold pending URLs
	currentCrawlingCh := make(chan int)
	finishCh := make(chan struct{})
	visitedLinksMap := sync.Map{}

	// Start the workers
	for i := 0; i < crawlerWorkerAmount; i++ {
		println("starting crawlerworker")
		go crawlerWorker(fmt.Sprintf("Worker #%d", i), c.fetcher, pendingURLch, currentCrawlingCh, &visitedLinksMap)
	}

	go func() {
		currentCrawling := 0
		for currentCrawl := range currentCrawlingCh {
			currentCrawling += currentCrawl
			fmt.Printf("[MONITOR]\tCurrently Crawling: %d\n", currentCrawling)
			if currentCrawling == 0 {
				fmt.Printf("[MONITOR]\tcurrentCrawling == 0. Closing channels.\n")
				close(pendingURLch)
				close(finishCh)
			}
		}
	}()

	// Add the starting URL to the pending URLs channel
	addLinksToPendingURLChannel("main", []string{urlToCrawl.String()}, pendingURLch, &visitedLinksMap)

	select {
	case <-time.After(time.Second * 30):
		linksCrawledAmount := 0
		visitedLinksMap.Range(func(key, value any) bool {
			linksCrawledAmount++
			fmt.Printf("key: %s, value: %s\n", key, value)
			return true
		})
		fmt.Printf("amount of links crawled: %d\n", linksCrawledAmount)
	}

	return nil, nil
}

func crawlerWorker(workerName string, fetcher fetcher.Fetcher, pendingURLch chan string, currentCrawling chan int, visitedLinksMap *sync.Map) {
	for linkToCrawl := range pendingURLch {
		println("inside range")
		currentCrawling <- 1
		fmt.Printf("[%s]\tStarting to crawl link: %s\n", workerName, linkToCrawl)
		visitedLinksMap.Store(linkToCrawl, true)

		parsedLink, err := url.Parse(linkToCrawl)
		if err != nil {
			currentCrawling <- -1
			fmt.Printf("[%s]\tcould not parse link [%s]. ignoring.\n", workerName, linkToCrawl)
			continue
		}
		fmt.Printf("[%s]\tCrawlWebpage start: %s\n", workerName, parsedLink)
		links, err := crawlWebpage(fetcher, *parsedLink)
		fmt.Printf("[%s]\tCrawlWebpage finish: %s.\n", workerName, parsedLink)
		if err != nil {
			currentCrawling <- -1
			fmt.Printf("[%s]\terror while crawling %s.", workerName, linkToCrawl)
			continue
		}

		fmt.Printf("[%s]\taddLinksToPendingURLChannel start.\n", workerName)
		addLinksToPendingURLChannel(workerName, links, pendingURLch, visitedLinksMap)
		fmt.Printf("[%s]\taddLinksToPendingURLChannel finish.\n", workerName)
		currentCrawling <- -1
	}
}

func addLinksToPendingURLChannel(workerName string, links []string, pendingURLch chan string, visitedLinksMap *sync.Map) {
	for _, link := range links {
		if _, ok := visitedLinksMap.Load(link); ok == false {
			fmt.Printf("[%s]\tADDING LINK TO CHANNEL: %s\n", workerName, link)
			pendingURLch <- link
			fmt.Printf("[%s]\tLINK ADDED TO CHANNEL: %s. CHANNEL LEN (pendingURLch): %d\n", workerName, link, len(pendingURLch))
		} else {
			fmt.Printf("[%s]\tLINK ALREADY EXISTS: %s\n", workerName, link)
		}
	}
	fmt.Printf("[%s]\tRETURNING...\n", workerName)
}

func crawlWebpage(httpFetcher fetcher.Fetcher, webpageURL url.URL) ([]string, error) {
	webpageContent, err := httpFetcher.GetWebpageContent(webpageURL)
	if err != nil {
		return nil, err
	}

	links, err := linkextractor.Extract(webpageURL, webpageContent)
	if err != nil {
		return nil, err
	}

	return links, nil
}
