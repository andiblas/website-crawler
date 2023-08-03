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
	defaultRecursionLimit  = 4
	defaultTimeout         = 15000
	defaultNumberOfRetries = 3
)

func main() {
	urlToCrawlArg := flag.String("url", "", "URL to crawl.")
	//recursionLimitArg := flag.Int("recursion_limit", defaultRecursionLimit, "Sets the amount of times the crawler will continue crawling on links found in a page. Must be greater than 0.")
	timeoutArg := flag.Int("timeout", defaultTimeout, "Please set the timeout in milliseconds. Must be greater than 0.")
	numberOfRetriesArg := flag.Int("retries", defaultNumberOfRetries, "Set the number of retries the crawler will try to fetch a page in case of errors. Must be 0 or greater than 0.")

	flag.Parse()

	timeout := validateTimeoutArg(*timeoutArg)
	//recursionLimit := validateRecursionLimit(*recursionLimitArg)
	parsedUrl := validateUrlToCrawl(*urlToCrawlArg)
	numberOfRetries := validateNumberOfRetries(*numberOfRetriesArg)

	httpFetcher := fetcher.NewHTTPFetcher(&http.Client{
		Timeout: time.Duration(timeout) * time.Millisecond,
	})

	//var concurrentCrawler crawler.Crawler
	//if numberOfRetries > 0 {
	//	backoffRetryFetcher := fetcher.NewExpBackoffRetryFetcher(httpFetcher, numberOfRetries, time.Second*4)
	//	concurrentCrawler = crawler.NewConcurrent(backoffRetryFetcher)
	//} else {
	//	concurrentCrawler = crawler.NewConcurrent(httpFetcher)
	//}

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

	//linkCount := 0
	//linkFoundCallback := func(link url.URL) {
	//	linkCount++
	//	fmt.Printf("[LINK %04d]\t%s\n", linkCount, link.String())
	//}

	// you can use the full crawling result set once finished
	//_, err := concurrentCrawler.Crawl(cancelCtx, parsedUrl, recursionLimit, linkFoundCallback)
	//if err != nil {
	//	fmt.Printf("%v\n", err)
	//}

	//fmt.Printf("Total links found: %d\n", linkCount)

	var anotherCrawler crawler.Crawler
	if numberOfRetries > 0 {
		backoffRetryFetcher := fetcher.NewExpBackoffRetryFetcher(httpFetcher, numberOfRetries, time.Second*4)
		anotherCrawler = crawler.NewBreadthFirstCrawler(backoffRetryFetcher)
	} else {
		anotherCrawler = crawler.NewBreadthFirstCrawler(httpFetcher)
	}

	anotherLinkFoundCallback := func(link url.URL) {
		fmt.Printf("[LINK] Crawling: %s\n", link.String())
	}
	anotherLinksFound, err := anotherCrawler.Crawl(cancelCtx, parsedUrl, 4, 5, anotherLinkFoundCallback)
	if err != nil {
		log.Fatalln(err)
	}
	for i, link := range anotherLinksFound {
		fmt.Printf("[ANOTHER LINK %04d]\t%s\n", i, link)
	}
	fmt.Printf("Another Total links found: %d\n", len(anotherLinksFound))
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

func validateRecursionLimit(recursionLimitArg int) int {
	if recursionLimitArg <= 0 {
		log.Fatalln("argument error: invalid recursion limit. must be greater than 0. example: --recursion_limit=2")
	}
	return recursionLimitArg
}

func validateTimeoutArg(timeoutArg int) int {
	if timeoutArg <= 0 {
		log.Fatalln("argument error: invalid timeout. example: --timeout=5000")
	}
	return timeoutArg
}

func validateNumberOfRetries(numberOfRetries int) int {
	if numberOfRetries < 0 {
		log.Fatalln("argument error: invalid retries argument. example: --retries=2")
	}
	return numberOfRetries
}
