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
	defaultDepth           = 4
	defaultMaxConcurrency  = 5
	defaultTimeout         = 15000
	defaultNumberOfRetries = 3
)

func main() {
	urlToCrawlArg := flag.String("url", "", "URL to crawl.")
	depthArg := flag.Int("depth", defaultDepth, "Sets the crawling depth. The depth is delimited by each time the crawler continues crawling on new discovered pages. Must be greater than 0.")
	maxConcurrencyArg := flag.Int("max_concurrency", defaultMaxConcurrency, "Sets the maximum concurrent requests the crawler can do. Must be greater than 0.")
	timeoutArg := flag.Int("timeout", defaultTimeout, "Please set the timeout in milliseconds. Must be greater than 0.")
	numberOfRetriesArg := flag.Int("retries", defaultNumberOfRetries, "Set the number of retries the crawler will try to fetch a page in case of errors. Must be 0 or greater than 0.")

	flag.Parse()

	timeout := validateTimeoutArg(*timeoutArg)
	depth := validateDepth(*depthArg)
	maxConcurrency := validateMaxConcurrency(*maxConcurrencyArg)
	parsedUrl := validateUrlToCrawl(*urlToCrawlArg)
	numberOfRetries := validateNumberOfRetries(*numberOfRetriesArg)

	httpFetcher := fetcher.NewHTTPFetcher(&http.Client{
		Timeout: time.Duration(timeout) * time.Millisecond,
	})

	ctx := context.Background()
	cancelCtx, cancelFunc := context.WithCancel(ctx)
	defer cancelFunc()

	go func() {
		// listen for interrupt signal
		interrupt := make(chan os.Signal, 1)
		signal.Notify(interrupt, os.Interrupt, syscall.SIGTERM)
		<-interrupt
		cancelFunc()
	}()

	var bfCrawler crawler.Crawler
	if numberOfRetries > 0 {
		backoffRetryFetcher := fetcher.NewExpBackoffRetryFetcher(httpFetcher, numberOfRetries, time.Second*4)
		bfCrawler = crawler.NewBreadthFirstCrawler(backoffRetryFetcher)
	} else {
		bfCrawler = crawler.NewBreadthFirstCrawler(httpFetcher)
	}

	errorCallback := func(link url.URL, err error) {
		fmt.Printf("[ERROR] error while crawling [%s] err: %v\n", link.String(), err)
	}
	linkFoundCb := func(link url.URL) {
		fmt.Printf("[LINK] Crawling: %s\n", link.String())
	}
	links, err := bfCrawler.Crawl(cancelCtx, parsedUrl, depth, maxConcurrency, linkFoundCb, errorCallback)

	if err != nil {
		log.Fatalln(err)
	}
	fmt.Printf("Total links found: %d\n", len(links))
}

func validateUrlToCrawl(urlToCrawlArg string) url.URL {
	errMessage := "argument error: invalid URL to crawl. example: --url=https://example.com"
	if strings.TrimSpace(urlToCrawlArg) == "" {
		log.Fatalln(errMessage)
	}

	parsedUrl, err := url.Parse(urlToCrawlArg)
	if err != nil {
		log.Fatalln(errMessage)
	}
	return *parsedUrl
}

func validateDepth(depthArg int) int {
	if depthArg <= 0 {
		log.Fatalln("argument error: invalid depth. must be greater than 0. example: --depth=2")
	}
	return depthArg
}

func validateMaxConcurrency(maxConcurrencyArg int) int {
	if maxConcurrencyArg <= 0 {
		log.Fatalln("argument error: invalid max_concurrency. must be greater than 0. example: --max_concurrency=2")
	}
	return maxConcurrencyArg
}

func validateTimeoutArg(timeoutArg int) int {
	if timeoutArg <= 0 {
		log.Fatalln("argument error: invalid timeout. example: --timeout=5000")
	}
	return timeoutArg
}

func validateNumberOfRetries(numberOfRetries int) int {
	if numberOfRetries < 0 {
		log.Fatalln("argument error: invalid retries. example: --retries=2")
	}
	return numberOfRetries
}
