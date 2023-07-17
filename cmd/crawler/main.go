package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/andiblas/website-crawler/internal/crawler"
	"github.com/andiblas/website-crawler/internal/fetcher"
)

const defaultTimeout = 10000

func main() {
	urlToCrawlArg := flag.String("url", "", "URL to crawl.")
	timeoutArg := flag.Int("timeout", defaultTimeout, "Please set the timeout in milliseconds. Default: 10000 milliseconds.")

	flag.Parse()

	timeout := validateTimeoutArg(*timeoutArg)
	parsedUrl := validateUrlToCrawl(*urlToCrawlArg)

	httpFetcher := fetcher.NewHTTPFetcher(&http.Client{
		Timeout: time.Duration(timeout) * time.Millisecond,
	})
	concurrentCrawler := crawler.NewConcurrent(httpFetcher)

	crawledLinks, err := concurrentCrawler.Crawl(parsedUrl)
	if err != nil {
		log.Fatalf("execution error: %v", err)
	}

	printResults(crawledLinks)
}

func printResults(crawledLinks []string) {
	fmt.Printf("[RESULTS] Total links found: %d\n", len(crawledLinks))
	for index, crawledLink := range crawledLinks {
		fmt.Printf("[Link #%d] %s\n", index, crawledLink)
	}
}

func validateUrlToCrawl(urlToCrawlArg string) url.URL {
	if strings.TrimSpace(urlToCrawlArg) == "" {
		log.Fatalln("argument error: invalid URL to crawl")
	}

	parsedUrl, err := url.Parse(urlToCrawlArg)
	if err != nil {
		log.Fatalln("argument error: invalid URL to crawl")
	}
	return *parsedUrl
}

func validateTimeoutArg(timeoutArg int) int {
	if timeoutArg <= 0 {
		log.Fatalln("argument error: invalid timeout")
	}
	return timeoutArg
}
