package linkextractor

import (
	"net/url"
	"strings"

	"golang.org/x/net/html"
)

func Extract(webpageURL url.URL, webpageContent string) ([]url.URL, error) {
	parsedHtmlContent, err := html.Parse(strings.NewReader(webpageContent))
	if err != nil {
		return nil, err
	}

	// I'm aware that the following links manipulation represents
	// a O(3n) operation and could be improved. I opted with this
	// approach since we are not going to deal with huge amounts
	// of links and also for the sake of readability.
	links := searchDomainMatchingLinks(webpageURL, parsedHtmlContent)
	linksWithoutDuplicates := removeDuplicates(links)
	normalizedLinks := normalizeLinks(linksWithoutDuplicates)

	return normalizedLinks, nil
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
				if parsedLink, matches := domainMatches(webpageURL, attr.Val); matches {
					links = append(links, parsedLink)
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

func normalizeLinks(links []url.URL) []url.URL {
	var normalizedLinks []url.URL
	for _, link := range links {
		normalizedLinks = append(normalizedLinks, Normalize(link))
	}
	return normalizedLinks
}

func domainMatches(webpageURL url.URL, hrefValue string) (url.URL, bool) {
	hrefUrl, err := url.Parse(hrefValue)
	if err != nil {
		return url.URL{}, false
	}

	return *hrefUrl, webpageURL.Host == Normalize(*hrefUrl).Host
}
