package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"golang.org/x/net/context"

	"github.com/andiblas/website-crawler/pkg/crawler"
	"github.com/andiblas/website-crawler/pkg/fetcher"
)

const (
	defaultPathDepth       = 2
	defaultTimeout         = 15000
	defaultNumberOfRetries = 3
)

func main() {
	urlToCrawlArg := flag.String("url", "", "URL to crawl.")
	pathDepthArg := flag.Int("path_depth", defaultPathDepth, "The path depth from the starting URL that you want to crawl. Must be greater than 0.")
	timeoutArg := flag.Int("timeout", defaultTimeout, "Please set the timeout in milliseconds. Must be greater than 0.")
	numberOfRetriesArg := flag.Int("retries", defaultNumberOfRetries, "Set the number of retries the crawler will try to fetch a page in case of errors. Must be 0 or greater than 0.")

	flag.Parse()

	timeout := validateTimeoutArg(*timeoutArg)
	pathDepth := validatePathDepth(*pathDepthArg)
	parsedUrl := validateUrlToCrawl(*urlToCrawlArg)
	numberOfRetries := validateNumberOfRetries(*numberOfRetriesArg)

	httpFetcher := fetcher.NewHTTPFetcher(&http.Client{
		Timeout: time.Duration(timeout) * time.Millisecond,
	})

	var concurrentCrawler crawler.Crawler
	if numberOfRetries > 0 {
		backoffRetryFetcher := fetcher.NewExpBackoffRetryFetcher(httpFetcher, numberOfRetries, time.Second*4)
		concurrentCrawler = crawler.NewConcurrent(backoffRetryFetcher)
	} else {
		concurrentCrawler = crawler.NewConcurrent(httpFetcher)
	}

	ctx := context.Background()
	cancelCtx, cancelFunc := context.WithCancel(ctx)

	go func() {
		// listen for interrupt signal
		interrupt := make(chan os.Signal, 1)
		signal.Notify(interrupt, os.Interrupt, syscall.SIGTERM)
		<-interrupt
		cancelFunc()
	}()

	crawledLinks, err := concurrentCrawler.Crawl(cancelCtx, parsedUrl, pathDepth)
	if err != nil {
		fmt.Printf("%v\n", err)
	}

	printResults(crawledLinks)
}

func printResults(crawledLinks []string) {
	fmt.Printf("[RESULTS] Links found: %d\n", len(crawledLinks))
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

func validatePathDepth(pathDepthArg int) int {
	if pathDepthArg <= 0 {
		log.Fatalln("argument error: invalid path depth")
	}
	return pathDepthArg
}

func validateTimeoutArg(timeoutArg int) int {
	if timeoutArg <= 0 {
		log.Fatalln("argument error: invalid timeout")
	}
	return timeoutArg
}

func validateNumberOfRetries(numberOfRetries int) int {
	if numberOfRetries < 0 {
		log.Fatalln("argument error: invalid retries argument")
	}
	return numberOfRetries
}
