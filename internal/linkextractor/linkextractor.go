package linkextractor

import (
	"net/url"
	"strings"

	"golang.org/x/net/html"
)

func Extract(webpageURL url.URL, webpageContent string) ([]string, error) {
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
