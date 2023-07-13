package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"

	"golang.org/x/net/html"
)

func main() {
	urlToCrawl, err := url.Parse("https://parserdigital.com/")
	if err != nil {
		log.Fatalf("url to crawl is invalid")
	}

	err = CrawlWebpage(urlToCrawl)
	if err != nil {
		log.Fatalf("error while crawling: %+v", err.Error())
	}
}

func CrawlWebpage(webpageURL *url.URL) error {
	webpageContent := GetWebpageContent(webpageURL)

	_, err := ExtractLinks(webpageURL, webpageContent)
	if err != nil {
		return err
	}

	return nil
}

func GetWebpageContent(url *url.URL) string {
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

func ExtractLinks(webpageURL *url.URL, webpageContent string) ([]string, error) {
	parsedHtmlContent, err := html.Parse(strings.NewReader(webpageContent))
	if err != nil {
		return nil, err
	}

	links := searchDomainMatchingLinks(webpageURL, parsedHtmlContent)
	linksWithoutDuplicates := removeDuplicates(links)

	for _, linksWithoutDuplicate := range linksWithoutDuplicates {
		fmt.Println(linksWithoutDuplicate)
	}

	return nil, nil
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

func searchDomainMatchingLinks(webpageURL *url.URL, node *html.Node) []string {
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

func domainMatches(webpageURL *url.URL, hrefValue string) bool {
	hrefUrl, err := url.Parse(hrefValue)
	if err != nil {
		return false
	}

	return webpageURL.Host == hrefUrl.Host
}
