package linkextractor

import (
	"io"
	"net/url"
	"strings"

	"golang.org/x/net/html"
)

// Extract extracts URLs from the given webpage content and returns a slice of normalized URLs.
// The function parses the HTML content of the webpage and searches for links within the same domain as the provided webpageURL.
func Extract(webpageURL url.URL, webpageContent io.Reader) ([]url.URL, error) {
	parsedHtmlContent, err := html.Parse(webpageContent)
	if err != nil {
		return nil, err
	}

	links := searchDomainMatchingLinks(webpageURL, parsedHtmlContent)
	linksWithoutDuplicates := removeDuplicates(links)

	return linksWithoutDuplicates, nil
}

func Normalize(urlToNormalize url.URL) url.URL {
	return url.URL{
		Scheme: urlToNormalize.Scheme,
		Host:   strings.Replace(urlToNormalize.Host, "www.", "", -1),
		Path:   strings.TrimRight(urlToNormalize.Path, "/"),
	}
}

func searchDomainMatchingLinks(webpageURL url.URL, node *html.Node) []url.URL {
	var links []url.URL
	if node.Type == html.ElementNode && node.Data == "a" {
		for _, attr := range node.Attr {
			if attr.Key == "href" {
				hrefUrl, err := url.Parse(attr.Val)
				if err != nil {
					continue
				}
				normalizedLink := handleRelativeLink(webpageURL, Normalize(*hrefUrl))
				if domainMatches(webpageURL, normalizedLink) {
					links = append(links, normalizedLink)
				}
			}
		}
	}

	for child := node.FirstChild; child != nil; child = child.NextSibling {
		links = append(links, searchDomainMatchingLinks(webpageURL, child)...)
	}

	return links
}

func removeDuplicates(links []url.URL) []url.URL {
	uniqueMap := make(map[string]bool)
	uniqueSlice := make([]url.URL, 0)

	for _, link := range links {
		if !uniqueMap[link.String()] {
			uniqueMap[link.String()] = true
			uniqueSlice = append(uniqueSlice, link)
		}
	}

	return uniqueSlice
}

func handleRelativeLink(baseLink url.URL, relativeLink url.URL) url.URL {
	if relativeLink.Host == "" || relativeLink.Scheme == "" {
		return url.URL{
			Scheme: baseLink.Scheme,
			Host:   baseLink.Host,
			Path:   relativeLink.Path,
		}
	}
	return relativeLink
}

func domainMatches(webpageURL url.URL, hrefValue url.URL) bool {
	return webpageURL.Host == hrefValue.Host || hrefValue.Host == ""
}
