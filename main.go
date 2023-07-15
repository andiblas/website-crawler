package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"golang.org/x/net/html"
)

const crawlerWorkerAmount = 20

func main() {
	urlToCrawl, err := url.Parse("https://parserdigital.com/")
	if err != nil {
		log.Fatalf("url to crawl is invalid")
	}

	pendingURLch := make(chan string) // Channel to hold pending URLs
	currentCrawlingCh := make(chan int)
	finishCh := make(chan struct{})
	visitedLinksMap := sync.Map{}

	// Start the workers
	for i := 0; i < crawlerWorkerAmount; i++ {
		println("starting crawlerworker")
		go crawlerWorker(fmt.Sprintf("Worker #%d", i), pendingURLch, currentCrawlingCh, &visitedLinksMap)
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
}

func crawlerWorker(workerName string, pendingURLch chan string, currentCrawling chan int, visitedLinksMap *sync.Map) {
	println("inside crawlerWorker")
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
		links, err := CrawlWebpage(*parsedLink)
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

func CrawlWebpage(webpageURL url.URL) ([]string, error) {
	webpageContent := GetWebpageContent(webpageURL)

	links, err := ExtractLinks(webpageURL, webpageContent)
	if err != nil {
		return nil, err
	}

	return links, nil
}

func GetWebpageContent(url url.URL) string {
	res, err := http.Get(url.String())
	if err != nil {
		log.Fatal(err)
	}
	content, err := io.ReadAll(res.Body)
	defer res.Body.Close()
	if err != nil {
		log.Fatal(err)
	}

	return string(content)
}

func ExtractLinks(webpageURL url.URL, webpageContent string) ([]string, error) {
	parsedHtmlContent, err := html.Parse(strings.NewReader(webpageContent))
	if err != nil {
		return nil, err
	}

	links := searchDomainMatchingLinks(webpageURL, parsedHtmlContent)
	// TODO normalize links
	linksWithoutDuplicates := removeDuplicates(links)

	return linksWithoutDuplicates, nil
}

func removeDuplicates(links []string) []string {
	uniqueMap := make(map[string]bool)
	uniqueSlice := make([]string, 0)

	for _, link := range links {
		if !uniqueMap[link] {
			uniqueMap[link] = true
			uniqueSlice = append(uniqueSlice, link)
		}
	}

	return uniqueSlice
}

func searchDomainMatchingLinks(webpageURL url.URL, node *html.Node) []string {
	var links []string
	if node.Type == html.ElementNode && node.Data == "a" {
		for _, attr := range node.Attr {
			if attr.Key == "href" && domainMatches(webpageURL, attr.Val) {
				links = append(links, attr.Val)
			}
		}
	}

	for child := node.FirstChild; child != nil; child = child.NextSibling {
		links = append(links, searchDomainMatchingLinks(webpageURL, child)...)
	}

	return links
}

func domainMatches(webpageURL url.URL, hrefValue string) bool {
	hrefUrl, err := url.Parse(hrefValue)
	if err != nil {
		return false
	}

	return webpageURL.Host == hrefUrl.Host
}
